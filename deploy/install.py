#!/usr/bin/env python3
"""Install a tested local image privately; never print or replace an existing token."""
import argparse
import os
from pathlib import Path
import re
import secrets
import subprocess

parser = argparse.ArgumentParser()
parser.add_argument("--revision", required=True)
args = parser.parse_args()
if not re.fullmatch(r"[0-9a-f]{40}", args.revision):
    parser.error("revision must be a full commit SHA")
os.chdir(Path(__file__).resolve().parent)
secret_dir = Path("secrets")
secret_dir.mkdir(mode=0o700, exist_ok=True)
secret_dir.chmod(0o700)
token = secret_dir / "kripa_api_token"
if not token.exists():
    # Directory is owner-only; bind-mounted file readable by unprivileged containers.
    fd = os.open(token, os.O_CREAT | os.O_EXCL | os.O_WRONLY, 0o444)
    with os.fdopen(fd, "w") as file:
        file.write(secrets.token_hex(32) + "\n")
    token.chmod(0o444)
image = "kripa:" + args.revision
subprocess.run(["docker", "image", "inspect", image], stdout=subprocess.DEVNULL, check=True)
env = dict(os.environ, KRIPA_IMAGE=image)
subprocess.run(["docker", "compose", "config", "--quiet"], env=env, check=True)
subprocess.run(["docker", "compose", "up", "-d", "--wait", "--wait-timeout", "90"], env=env, check=True)
print("KRIPA private deployment started; run authenticated smoke and verify Prometheus target.")
