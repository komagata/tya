class Tya < Formula
  desc "Small indentation-based dynamic language"
  homepage "https://github.com/komagata/tya"
  url "https://github.com/komagata/tya/archive/refs/tags/v0.62.0.tar.gz"
  sha256 "REPLACE_AFTER_TAG_PUSH"
  license "MIT"
  head "https://github.com/komagata/tya.git", branch: "main"

  depends_on "go" => :build
  depends_on "zig"

  def install
    system "go", "build", *std_go_args(output: bin/"tya"), "./cmd/tya"
    (pkgshare/"runtime").install Dir["runtime/*"]
    (pkgshare/"stdlib").install Dir["stdlib/*"]
  end

  test do
    (testpath/"hello.tya").write <<~TYA
      print("Hello, Tya")
      print("  ".blank?())
    TYA

    assert_equal "0.62.0\n", shell_output("#{bin}/tya version")
    assert_equal "Hello, Tya\ntrue\n", shell_output("#{bin}/tya run #{testpath}/hello.tya")

    # v0.49: `tya new` scaffolds a minimal project tree.
    cd testpath do
      system bin/"tya", "new", "scaffold"
      assert_predicate testpath/"scaffold/tya.toml", :exist?
      assert_predicate testpath/"scaffold/src/main.tya", :exist?
      assert_predicate testpath/"scaffold/.gitignore", :exist?
    end

    # v0.49: `tya task` lists tasks defined in tya.toml.
    cd testpath/"scaffold" do
      output = shell_output("#{bin}/tya task")
      assert_match "run", output
    end

    # v0.49: `tya lint` reports unused locals on dirty sources.
    (testpath/"dirty.tya").write <<~TYA
      x = 1
      y = 2
      print(x)
    TYA
    assert_match "TYAL0001", shell_output("#{bin}/tya lint #{testpath}/dirty.tya", 1)

    # v0.52: `tya lsp --help` prints usage and exits 0.
    assert_match "tya lsp", shell_output("#{bin}/tya lsp --help")

    # v0.51: `tya doc` walks src/ and reports top-level bindings.
    (testpath/"docproj/src").mkpath
    (testpath/"docproj/src/lib.tya").write <<~TYA
      # Returns the doubled value.
      double = x -> x * 2
    TYA
    cd testpath/"docproj" do
      assert_match "function double", shell_output("#{bin}/tya doc")
    end
  end
end
