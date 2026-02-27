#!/bin/bash
# Build release binaries for all platforms

set -e

VERSION="${1:-v0.0.1}"

echo "🏗️  Building Vectra Guard $VERSION"
echo "=================================="
echo ""

# Create dist directory
rm -rf dist
mkdir -p dist

# Platforms to build
PLATFORMS=(
    "darwin/amd64"
    "darwin/arm64"
    "linux/amd64"
    "linux/arm64"
    "windows/amd64"
    "windows/arm64"
)

echo "Building for platforms:"
for platform in "${PLATFORMS[@]}"; do
    echo "  - $platform"
done
echo ""

# Build for each platform
for platform in "${PLATFORMS[@]}"; do
    GOOS="${platform%/*}"
    GOARCH="${platform#*/}"
    OUTPUT="dist/vectra-guard-${GOOS}-${GOARCH}"
    
    if [ "$GOOS" = "windows" ]; then
        OUTPUT="${OUTPUT}.exe"
    fi
    
    echo "📦 Building $GOOS/$GOARCH..."
    GOOS=$GOOS GOARCH=$GOARCH go build -ldflags="-s -w -X github.com/vectra-guard/vectra-guard/cmd.Version=$VERSION" -o "$OUTPUT" main.go
    
    # Calculate checksum
    if command -v shasum &> /dev/null; then
        shasum -a 256 "$OUTPUT" | tee -a dist/checksums.txt
    fi
done

echo ""
echo "✅ Build complete!"
echo ""

# Create checksums
echo "🔐 Generating checksums..."
cd dist
shasum -a 256 vectra-guard-* > checksums.txt 2>/dev/null || true
cd ..

echo ""
echo "✅ Release binaries ready!"
echo ""
echo "📦 Binaries in dist/:"
ls -lh dist/vectra-guard-*
echo ""
echo "📝 Next steps:"
echo "   1. Create GitHub release: https://github.com/xadnavyaai/vectra-guard/releases/new"
echo "   2. Upload these files from dist/ folder:"
echo "      • vectra-guard-darwin-amd64"
echo "      • vectra-guard-darwin-arm64"
echo "      • vectra-guard-linux-amd64"
echo "      • vectra-guard-linux-arm64"
echo "      • vectra-guard-windows-amd64.exe"
echo "      • vectra-guard-windows-arm64.exe"
echo "      • checksums.txt"
echo "   3. Publish release"
echo ""

