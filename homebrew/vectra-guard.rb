# Homebrew Formula for Vectra Guard
# To use: 
#   1. Create a tap: https://github.com/xadnavyaai/homebrew-tap
#   2. Add this file to Formula/vectra-guard.rb
#   3. Users install with: brew install xadnavyaai/tap/vectra-guard

class VectraGuard < Formula
  desc "Security guard for AI coding agents and development workflows"
  homepage "https://github.com/xadnavyaai/vectra-guard"
  url "https://github.com/xadnavyaai/vectra-guard/archive/v1.0.0.tar.gz"
  sha256 :no_check
  license "Apache-2.0"
  head "https://github.com/xadnavyaai/vectra-guard.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "./main.go"
    
    # Install binary
    bin.install "vectra-guard"
    
    # Install scripts
    (prefix/"scripts").install Dir["scripts/*"]
    
    # Install documentation
    doc.install "README.md", "GETTING_STARTED.md", "Project.md"
  end

  def post_install
    ohai "Vectra Guard installed successfully!"
    ohai ""
    ohai "Get started:"
    ohai "  vectra-guard init                # Initialize global config (~/.config/vectra-guard/)"
    ohai ""
    ohai "Install universal protection (recommended):"
    ohai "  #{prefix}/scripts/install-shell-tracker.sh"
    ohai ""
    ohai "Documentation: #{doc}"
  end

  test do
    # Test that binary runs
    assert_match "usage:", shell_output("#{bin}/vectra-guard 2>&1", 1)
    
    # Test init command
    system "#{bin}/vectra-guard", "init"
    assert_predicate testpath/"vectra-guard.yaml", :exist?
  end
end
