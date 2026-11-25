#!/usr/bin/env python3
"""Build self-contained Windows installers from the pinned launcher lockfile."""

import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
from urllib.request import urlopen
import zipfile

SOURCE = Path(__file__).resolve().parent
LAUNCHER = SOURCE.parent
OUT = SOURCE / "dist"
NODE_VERSION = "24.22.3"
TARGETS = {
    "x64": ("amd64", "86fc4785327db5f5c7558dc6156c543b433c313e813a222e088086222ab4afa6"),
    "arm64": ("arm64", "172e320db179ce416fc012b96b9917dacfb442beb081de57c4deaed7b5a24644"),
}


def fetch_node(architecture, expected, temp):
    node = temp / "node.exe"
    url = f"https://nodejs.org/dist/v{NODE_VERSION}/win-{architecture}/node.exe"
    with urlopen(url, timeout=120) as response, node.open("wb") as file:
        shutil.copyfileobj(response, file)
    if sha256(node) != expected:
        raise ValueError(f"Node.js checksum mismatch for {architecture}")
    return node


def sha256(path):
    with path.open("rb") as file:
        return hashlib.file_digest(file, "sha256").hexdigest()


def add(archive, path, name):
    entry = zipfile.ZipInfo(name)
    entry.compress_type = zipfile.ZIP_DEFLATED
    entry.create_system = 3
    entry.external_attr = 0o100644 << 16
    with archive.open(entry, "w") as output, path.open("rb") as source:
        shutil.copyfileobj(source, output)


def main():
    OUT.mkdir(exist_ok=True)
    npm = shutil.which("npm.cmd") or shutil.which("npm")
    if not npm:
        raise RuntimeError("npm is needed on the build machine.")
    with tempfile.TemporaryDirectory(prefix="web-chromium-") as work:
        work = Path(work)
        runtime = work / "runtime"
        runtime.mkdir()
        for name in ("package.json", "package-lock.json", "run.mjs"):
            shutil.copyfile(LAUNCHER / name, runtime / name)
        subprocess.run(
            [npm, "ci", "--ignore-scripts", "--no-audit", "--no-fund"],
            cwd=runtime,
            check=True,
        )
        license_file = runtime / "Node-LICENSE.txt"
        with urlopen(f"https://raw.githubusercontent.com/nodejs/node/v{NODE_VERSION}/LICENSE", timeout=30) as response:
            license_file.write_bytes(response.read())
        build = work / "build"
        build.mkdir()
        for path in [*SOURCE.glob("*.go"), SOURCE / "go.mod", SOURCE / "go.sum"]:
            shutil.copyfile(path, build / path.name)
        subprocess.run(["go", "test", "./..."], cwd=build, check=True)
        goroot = Path(subprocess.check_output(["go", "env", "GOROOT"], cwd=build, text=True).strip())
        shutil.copyfile(goroot / "LICENSE", runtime / "Go-LICENSE.txt")
        windows_module = json.loads(subprocess.check_output(
            ["go", "mod", "download", "-json", "golang.org/x/sys"], cwd=build, text=True,
        ))
        shutil.copyfile(Path(windows_module["Dir"]) / "LICENSE", runtime / "x-sys-LICENSE.txt")
        checksums = []
        for label, (goarch, expected) in TARGETS.items():
            node = fetch_node(label, expected, work)
            with zipfile.ZipFile(build / "runtime.zip", "w") as archive:
                add(archive, node, "node.exe")
                for path in sorted(runtime.rglob("*")):
                    if path.is_file() and not path.is_symlink():
                        name = path.relative_to(runtime).as_posix()
                        if not name.startswith("node_modules/.bin/"):
                            add(archive, path, name)
            output = OUT / f"Web-Chromium-MCP-Setup-{label}.exe"
            env = dict(os.environ, GOOS="windows", GOARCH=goarch, CGO_ENABLED="0")
            subprocess.run(
                ["go", "build", "-trimpath", "-ldflags=-s -w -H=windowsgui", "-o", output, "."],
                cwd=build,
                env=env,
                check=True,
            )
            checksums.append(f"{sha256(output)}  {output.name}")
        (OUT / "SHA256SUMS.txt").write_text("\n".join(checksums) + "\n", encoding="ascii")


if __name__ == "__main__":
    main()
