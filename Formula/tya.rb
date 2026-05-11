class Tya < Formula
  desc "Small indentation-based dynamic language"
  homepage "https://github.com/komagata/tya"
  url "https://github.com/komagata/tya/archive/refs/tags/v0.49.0.tar.gz"
  sha256 "REPLACE_AFTER_TAG_PUSH"
  license "MIT"
  head "https://github.com/komagata/tya.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(output: bin/"tya"), "./cmd/tya"
    (pkgshare/"runtime").install Dir["runtime/*"]
    (pkgshare/"stdlib").install Dir["stdlib/*"]
  end

  test do
    (testpath/"hello.tya").write <<~TYA
      import string

      print "Hello, Tya"
      print string.blank("  ")
    TYA
    (testpath/"hello_test.tya").write <<~TYA
      assert true
    TYA

    assert_equal "0.49.0\n", shell_output("#{bin}/tya version")
    assert_equal "Hello, Tya\ntrue\n", shell_output("#{bin}/tya run #{testpath}/hello.tya")
    assert_empty shell_output("#{bin}/tya test #{testpath}/hello_test.tya")

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
  end
end
