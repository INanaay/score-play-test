import argparse
import base64
import hashlib
import json
import mimetypes
import os
import sys
import requests
from datetime import datetime
from urllib.parse import urlparse

BASE_URL = os.environ.get("API_URL", "http://localhost:8080/api/v1")

def get_sha256(file_path):
    sha256_hash = hashlib.sha256()
    with open(file_path, "rb") as f:
        for byte_block in iter(lambda: f.read(4096), b""):
            sha256_hash.update(byte_block)
    return base64.b64encode(sha256_hash.digest()).decode('utf-8')

def get_content_type(file_path):
    mime_type, _ = mimetypes.guess_type(file_path)
    return mime_type or "application/octet-stream"



def tag_bulk_create(args):
    url = f"{BASE_URL}/tag"
    resp = requests.post(url, json={"tags": args.tags})
    if resp.status_code == 201:
        print(f"Tags {args.tags} created successfully.")
    else:
        print(f"Error: {resp.status_code} - {resp.text}")

def tag_list(args):
    url = f"{BASE_URL}/tag/"
    limit = args.limit
    marker = args.marker
    
    while True:
        params = {"limit": limit}
        if marker:
            params["marker"] = marker
        
        resp = requests.get(url, params=params)
        if resp.status_code == 200:
            data = resp.json()
            print(json.dumps(data, indent=2))
            
            marker = data.get("nextMarker")
            if not args.auto or not marker:
                break
            print(f"--- Fetching next page with marker: {marker} ---")
        else:
            print(f"Error: {resp.status_code} - {resp.text}")
            break

def file_request_upload(args):
    file_size = os.path.getsize(args.path)
    checksum = get_sha256(args.path)
    filename = os.path.basename(args.path)
    content_type = get_content_type(args.path)

    url = f"{BASE_URL}/file/upload"
    payload = {
        "filename": filename,
        "content_type": content_type,
        "size_bytes": file_size,
        "checksum_sha256": checksum,
        "tags": args.tags
    }
    resp = requests.post(url, json=payload)
    if resp.status_code == 201:
        data = resp.json()
        print(json.dumps(data, indent=2))
        return data
    else:
        print(f"Error: {resp.status_code} - {resp.text}")
        return None

def fix_presigned_url(url):
    # If the API is running in Docker, it might return 'minio' as the host.
    # If we are running the client from outside Docker, we need to use 'localhost'.
    s3_host = os.environ.get("S3_HOST")
    if s3_host:
        # Example: S3_HOST="localhost:9000"
        import re
        return re.sub(r'http[s]?://[^/]+', f'http://{s3_host}', url)
    
    # Simple default replacement if 'minio' is found and not resolvable
    if "://minio:" in url:
        return url.replace("://minio:", "://localhost:")
    return url

def file_upload_simple(args):
    data = file_request_upload(args)
    if not data:
        return

    original_url = data["presigned_url"]
    presigned_url = fix_presigned_url(original_url)
    headers = data.get("headers", {})
    
    # If the host was changed (e.g. minio -> localhost), we must preserve 
    # the original Host header because it's part of the S3 signature.
    original_host = urlparse(original_url).netloc
    if urlparse(presigned_url).netloc != original_host:
        headers["Host"] = original_host

    print(f"Uploading {args.path} to {presigned_url}...")
    with open(args.path, "rb") as f:
        resp = requests.put(presigned_url, data=f, headers=headers)
    
    if resp.status_code == 200:
        print("Upload successful!")
        print(f"File ID: {data['file_id']}")
    else:
        print(f"Upload failed: {resp.status_code} - {resp.text}")

def file_request_multipart(args):
    file_size = os.path.getsize(args.path)
    checksum = get_sha256(args.path)
    filename = os.path.basename(args.path)
    content_type = get_content_type(args.path)

    url = f"{BASE_URL}/file/upload/multipart"
    payload = {
        "filename": filename,
        "content_type": content_type,
        "size_bytes": file_size,
        "checksum_sha256": checksum,
        "tags": args.tags
    }
    resp = requests.post(url, json=payload)
    if resp.status_code == 201:
        data = resp.json()
        print(json.dumps(data, indent=2))
        return data
    else:
        print(f"Error: {resp.status_code} - {resp.text}")
        return None

def file_upload_multipart(args):
    init_data = file_request_multipart(args)
    if not init_data:
        return

    session_id = init_data["session_id"]
    part_size = init_data["part_size"]
    file_path = args.path
    file_size = os.path.getsize(file_path)

    completed_parts = []
    
    with open(file_path, "rb") as f:
        part_number = 1
        while True:
            chunk = f.read(part_size)
            if not chunk:
                break
            
            chunk_checksum = base64.b64encode(hashlib.sha256(chunk).digest()).decode('utf-8')
            chunk_length = len(chunk)
            
            # Request presigned URL for this part
            url = f"{BASE_URL}/file/upload/multipart/{session_id}/parts"
            payload = {
                "parts": [
                    {
                        "part_number": part_number,
                        "checksum": chunk_checksum,
                        "content_length": chunk_length
                    }
                ]
            }
            resp = requests.post(url, json=payload)
            if resp.status_code != 201:
                print(f"Failed to get presigned URL for part {part_number}: {resp.text}")
                return
            
            presigned_data = resp.json()["presigned_parts"][0]
            original_part_url = presigned_data["presigned_url"]
            presigned_url = fix_presigned_url(original_part_url)
            headers = presigned_data.get("headers", {})
            
            # Preserve original Host for signature
            original_host = urlparse(original_part_url).netloc
            if urlparse(presigned_url).netloc != original_host:
                headers["Host"] = original_host

            print(f"Uploading part {part_number} ({chunk_length} bytes)...")
            put_resp = requests.put(presigned_url, data=chunk, headers=headers)
            if put_resp.status_code != 200:
                print(f"Failed to upload part {part_number}: {put_resp.status_code} - {put_resp.text}")
                return
            
            etag = put_resp.headers.get("ETag")
            if not etag:
                 print(f"Warning: No ETag returned for part {part_number}")
                 # Minio usually returns ETag in quotes
            
            completed_parts.append({
                "part_number": part_number,
                "etag": etag.strip('"') if etag else "",
                "checksum": chunk_checksum
            })
            
            part_number += 1

    # Complete multipart upload
    print("Completing multipart upload...")
    url = f"{BASE_URL}/file/upload/multipart/{session_id}/complete"
    resp = requests.post(url, json={"parts": completed_parts})
    if resp.status_code == 201:
        data = resp.json()
        print("Multipart upload successful!")
        print(f"File ID: {data['file_id']}")
    else:
        print(f"Failed to complete multipart upload: {resp.status_code} - {resp.text}")

def file_test_resume_flow(args):
    # 1. Initiate Upload
    init_data = file_request_multipart(args)
    if not init_data:
        return

    session_id = init_data["session_id"]
    part_size = init_data["part_size"]
    file_path = args.path
    file_size = os.path.getsize(file_path)
    total_parts = (file_size + part_size - 1) // part_size
    
    stop_after = args.stop_after if args.stop_after else total_parts // 2
    if stop_after < 1: stop_after = 1
    
    print(f"\n=== STARTING UPLOAD (Simulating failure after {stop_after} parts) ===")
    
    # 2. Upload Partial
    with open(file_path, "rb") as f:
        part_number = 1
        while part_number <= stop_after:
            chunk = f.read(part_size)
            if not chunk:
                break
            
            chunk_checksum = base64.b64encode(hashlib.sha256(chunk).digest()).decode('utf-8')
            chunk_length = len(chunk)
            
            url = f"{BASE_URL}/file/upload/multipart/{session_id}/parts"
            payload = {
                "parts": [
                    {
                        "part_number": part_number,
                        "checksum": chunk_checksum,
                        "content_length": chunk_length
                    }
                ]
            }
            resp = requests.post(url, json=payload)
            if resp.status_code != 201:
                print(f"Failed to get presigned URL for part {part_number}: {resp.text}")
                return
            
            presigned_data = resp.json()["presigned_parts"][0]
            original_part_url = presigned_data["presigned_url"]
            presigned_url = fix_presigned_url(original_part_url)
            headers = presigned_data.get("headers", {})
            
            original_host = urlparse(original_part_url).netloc
            if urlparse(presigned_url).netloc != original_host:
                headers["Host"] = original_host

            print(f"Uploading part {part_number}...")
            put_resp = requests.put(presigned_url, data=chunk, headers=headers)
            if put_resp.status_code != 200:
                print(f"Failed to upload part {part_number}: {put_resp.status_code} - {put_resp.text}")
                return
            part_number += 1

    print("\n=== PAUSED (Simulated Crash) ===")
    print("Verifying uploaded parts with list-parts...")
    
    # 3. List Parts
    existing_parts = {} 
    marker = 0
    while True:
        url = f"{BASE_URL}/file/upload/multipart/{session_id}/parts"
        params = {"nb_parts": 1000}
        if marker:
            params["marker"] = marker
            
        resp = requests.get(url, params=params)
        if resp.status_code != 200:
            print(f"Failed to list parts: {resp.status_code} - {resp.text}")
            return
            
        data = resp.json()
        for p in data.get("parts", []):
            existing_parts[p["part_number"]] = p["etag"]
            
        marker = data.get("parts_marker")
        if not marker or marker == 0:
            break

    print(f"Server confirmed {len(existing_parts)} uploaded parts.")

    # 4. Resume
    print("\n=== RESUMING UPLOAD ===")
    completed_parts = []
    
    with open(file_path, "rb") as f:
        part_number = 1
        while True:
            chunk = f.read(part_size)
            if not chunk:
                break
            
            chunk_checksum = base64.b64encode(hashlib.sha256(chunk).digest()).decode('utf-8')
            
            if part_number in existing_parts:
                print(f"Skipping part {part_number} (verified on server)")
                completed_parts.append({
                    "part_number": part_number,
                    "etag": existing_parts[part_number],
                    "checksum": chunk_checksum
                })
            else:
                chunk_length = len(chunk)
                url = f"{BASE_URL}/file/upload/multipart/{session_id}/parts"
                payload = {
                    "parts": [
                        {
                            "part_number": part_number,
                            "checksum": chunk_checksum,
                            "content_length": chunk_length
                        }
                    ]
                }
                resp = requests.post(url, json=payload)
                if resp.status_code != 201:
                    print(f"Failed to get presigned URL for part {part_number}: {resp.text}")
                    return
                
                presigned_data = resp.json()["presigned_parts"][0]
                original_part_url = presigned_data["presigned_url"]
                presigned_url = fix_presigned_url(original_part_url)
                headers = presigned_data.get("headers", {})
                
                original_host = urlparse(original_part_url).netloc
                if urlparse(presigned_url).netloc != original_host:
                    headers["Host"] = original_host

                print(f"Uploading part {part_number}...")
                put_resp = requests.put(presigned_url, data=chunk, headers=headers)
                if put_resp.status_code != 200:
                    print(f"Failed to upload part {part_number}: {put_resp.status_code} - {put_resp.text}")
                    return
                
                etag = put_resp.headers.get("ETag")
                completed_parts.append({
                    "part_number": part_number,
                    "etag": etag.strip('"') if etag else "",
                    "checksum": chunk_checksum
                })
            
            part_number += 1

    # Complete
    print("Completing multipart upload...")
    url = f"{BASE_URL}/file/upload/multipart/{session_id}/complete"
    resp = requests.post(url, json={"parts": completed_parts})
    if resp.status_code == 201:
        data = resp.json()
        print("Multipart upload successful!")
        print(f"File ID: {data['file_id']}")
    else:
        print(f"Failed to complete multipart upload: {resp.status_code} - {resp.text}")


def file_test_bad_checksum(args):
    print("=== TESTING BAD CHECKSUM UPLOAD ===")
    # 1. Initiate Upload
    init_data = file_request_multipart(args)
    if not init_data:
        return

    session_id = init_data["session_id"]
    part_size = init_data["part_size"]
    file_path = args.path
    
    # 2. Prepare Part 1 (Correct Data)
    with open(file_path, "rb") as f:
        chunk = f.read(part_size)
    
    if not chunk:
        print("Error: Empty file")
        return

    # Calculate correct checksum for signing
    correct_checksum = base64.b64encode(hashlib.sha256(chunk).digest()).decode('utf-8')
    chunk_length = len(chunk)

    print(f"1. Requesting presigned URL for Part 1 with CORRECT checksum: {correct_checksum}")
    
    # 3. Get Presigned URL
    url = f"{BASE_URL}/file/upload/multipart/{session_id}/parts"
    payload = {
        "parts": [
            {
                "part_number": 1,
                "checksum": correct_checksum,
                "content_length": chunk_length
            }
        ]
    }
    resp = requests.post(url, json=payload)
    if resp.status_code != 201:
        print(f"Failed to get presigned URL: {resp.text}")
        return

    presigned_data = resp.json()["presigned_parts"][0]
    presigned_url = fix_presigned_url(presigned_data["presigned_url"])
    headers = presigned_data.get("headers", {})
    
    # Fix Host header if needed
    original_part_url = presigned_data["presigned_url"]
    original_host = urlparse(original_part_url).netloc
    if urlparse(presigned_url).netloc != original_host:
        headers["Host"] = original_host

    # 4. Corrupt the data
    print("2. Corrupting data (changing last byte)...")
    corrupted_chunk = bytearray(chunk)
    if len(corrupted_chunk) > 0:
        corrupted_chunk[-1] = (corrupted_chunk[-1] + 1) % 256
    
    # 5. Upload Corrupted Data with Original Headers
    print(f"3. Uploading CORRUPTED data to: {presigned_url}")
    print(f"   Headers: {headers}")
    
    put_resp = requests.put(presigned_url, data=corrupted_chunk, headers=headers)
    
    print("\n=== RESULT ===")
    print(f"Status Code: {put_resp.status_code}")
    print(f"Response Body: {put_resp.text}")
    
    if put_resp.status_code == 400:
        print("\nSUCCESS: MinIO rejected the corrupted part as expected.")
    else:
        print("\nFAILURE: Upload was NOT rejected as expected (or rejected with unexpected error).")


def file_get(args):
    url = f"{BASE_URL}/file/{args.file_id}/"
    resp = requests.get(url)
    if resp.status_code == 200:
        data = resp.json()
        data["url"] = fix_presigned_url(data["url"])
        print(json.dumps(data, indent=2))
    else:
        print(f"Error: {resp.status_code} - {resp.text}")

def file_list_parts(args):
    url = f"{BASE_URL}/file/upload/multipart/{args.session_id}/parts"
    nb_parts = args.nb_parts
    marker = args.marker

    while True:
        params = {"nb_parts": nb_parts}
        if marker:
            params["marker"] = marker

        resp = requests.get(url, params=params)
        if resp.status_code == 200:
            data = resp.json()
            print(json.dumps(data, indent=2))

            marker = data.get("parts_marker")
            # If marker is 0 or -1 (depending on API), it might mean end.
            # Looking at your Go code, it returns the next marker.
            # If no more parts, it usually returns a marker that doesn't change or 0.
            if not args.auto or not marker or marker == -1:
                break
            print(f"--- Fetching next page with marker: {marker} ---")
        else:
            print(f"Error: {resp.status_code} - {resp.text}")
            break

def file_download(args):
    # First get the file info to get the presigned URL
    url = f"{BASE_URL}/file/{args.file_id}/"
    resp = requests.get(url)
    if resp.status_code != 200:
        print(f"Error getting file info: {resp.status_code} - {resp.text}")
        return

    data = resp.json()
    presigned_url = fix_presigned_url(data["url"])
    filename = data["filename"]

    download_path = f"downloaded_{filename}"
    print(f"Downloading {filename} to {download_path}...")

    # Preserve original Host for signature if host was fixed
    headers = {}
    original_host = urlparse(data["url"]).netloc
    if urlparse(presigned_url).netloc != original_host:
        headers["Host"] = original_host

    with requests.get(presigned_url, headers=headers, stream=True) as r:
        r.raise_for_status()
        with open(download_path, 'wb') as f:
            for chunk in r.iter_content(chunk_size=8192):
                f.write(chunk)

    print(f"Download complete: {download_path}")

def main():
    parser = argparse.ArgumentParser(description="Score-Play API Client")
    subparsers = parser.add_subparsers(dest="command", help="Commands")

    # Tag commands
    tag_parser = subparsers.add_parser("tag", help="Tag operations")
    tag_subparsers = tag_parser.add_subparsers(dest="subcommand")


    bulk_tag_p = tag_subparsers.add_parser("create", help="create tags")
    bulk_tag_p.add_argument("tags", nargs="+", help="Tag names")
    bulk_tag_p.set_defaults(func=tag_bulk_create)

    list_tag_p = tag_subparsers.add_parser("list", help="List tags")
    list_tag_p.add_argument("--limit", type=int, default=10)
    list_tag_p.add_argument("--marker", help="Marker for pagination")
    list_tag_p.add_argument("--auto", action="store_true", help="Automatically follow markers")
    list_tag_p.set_defaults(func=tag_list)

    # File commands
    file_parser = subparsers.add_parser("file", help="File operations")
    file_subparsers = file_parser.add_subparsers(dest="subcommand")

    req_upload_p = file_subparsers.add_parser("request-upload", help="Request a simple upload")
    req_upload_p.add_argument("path", help="Path to file")
    req_upload_p.add_argument("tags", nargs="+", help="Tags for the file")
    req_upload_p.set_defaults(func=file_request_upload)

    upload_simple_p = file_subparsers.add_parser("upload", help="Perform a simple upload")
    upload_simple_p.add_argument("path", help="Path to file")
    upload_simple_p.add_argument("tags", nargs="+", help="Tags for the file")
    upload_simple_p.set_defaults(func=file_upload_simple)

    req_multi_p = file_subparsers.add_parser("request-multipart", help="Request a multipart upload")
    req_multi_p.add_argument("path", help="Path to file")
    req_multi_p.add_argument("tags", nargs="+", help="Tags for the file")
    req_multi_p.set_defaults(func=file_request_multipart)

    upload_multi_p = file_subparsers.add_parser("upload-multipart", help="Perform a multipart upload")
    upload_multi_p.add_argument("path", help="Path to file")
    upload_multi_p.add_argument("tags", nargs="+", help="Tags for the file")
    upload_multi_p.set_defaults(func=file_upload_multipart)

    # NEW: Test Resume Flow
    test_resume_p = file_subparsers.add_parser("test-resume-flow", help="Test upload -> pause -> list -> resume flow")
    test_resume_p.add_argument("path", help="Path to file")
    test_resume_p.add_argument("tags", nargs="+", help="Tags for the file")
    test_resume_p.add_argument("--stop-after", type=int, help="Stop after this many parts")
    test_resume_p.set_defaults(func=file_test_resume_flow)

    # NEW: Test Bad Checksum
    test_bad_checksum_p = file_subparsers.add_parser("test-bad-checksum", help="Test upload with mismatched checksum")
    test_bad_checksum_p.add_argument("path", help="Path to file")
    test_bad_checksum_p.add_argument("tags", nargs="+", help="Tags for the file")
    test_bad_checksum_p.set_defaults(func=file_test_bad_checksum)

    get_file_p = file_subparsers.add_parser("get", help="Get file info")
    get_file_p.add_argument("file_id", help="UUID of the file")
    get_file_p.set_defaults(func=file_get)

    download_file_p = file_subparsers.add_parser("download", help="Download a file")
    download_file_p.add_argument("file_id", help="UUID of the file")
    download_file_p.set_defaults(func=file_download)

    list_parts_p = file_subparsers.add_parser("list-parts", help="List uploaded parts for a session")
    list_parts_p.add_argument("session_id", help="Session UUID")
    list_parts_p.add_argument("nb_parts", type=int, help="Number of parts to list")
    list_parts_p.add_argument("--marker", help="Marker for pagination")
    list_parts_p.add_argument("--auto", action="store_true", help="Automatically follow markers")
    list_parts_p.set_defaults(func=file_list_parts)

    args = parser.parse_args()
    if not args.command:
        parser.print_help()
        return

    if hasattr(args, "func"):
        args.func(args)
    else:
        # If subcommand is not provided for tag/file
        if args.command == "tag":
            tag_parser.print_help()
        elif args.command == "file":
            file_parser.print_help()

if __name__ == "__main__":
    main()
