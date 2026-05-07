class Tya < Formula
  desc "Small indentation-based dynamic language"
  homepage "https://github.com/komagata/tya"
  head "https://github.com/komagata/tya.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(output: bin/"tya"), "./cmd/tya"
    (pkgshare/"runtime").install Dir["runtime/*"]
  end

  test do
    (testpath/"hello.tya").write <<~TYA
      print "Hello, Tya"
    TYA

    assert_equal "0.2.0\n", shell_output("#{bin}/tya version")
    assert_equal "Hello, Tya\n", shell_output("#{bin}/tya run #{testpath}/hello.tya")
  end
end
