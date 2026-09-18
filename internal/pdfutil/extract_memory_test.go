package pdfutil

import (
	"bytes"
	"context"
	"image"
	"image/draw"
	"image/png"
	"os"
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ExtractText in memory", func() {
	var fakePDF []byte
	ctx := context.Background()

	BeforeEach(func() {
		var err error
		fakePDF, err = os.ReadFile("testdata/fake_statement.pdf")
		Expect(err).NotTo(HaveOccurred())
	})

	// Copy per call: ExtractText zeroes the buffer it was given.
	copyOfFakePDF := func() []byte { return bytes.Clone(fakePDF) }

	tempDirAssertion := func() string {
		tmp := GinkgoT().TempDir()
		GinkgoT().Setenv("TMPDIR", tmp)
		return tmp
	}

	expectNoTempFiles := func(tmp string) {
		entries, err := os.ReadDir(tmp)
		Expect(err).NotTo(HaveOccurred())
		Expect(entries).To(BeEmpty())
	}

	It("extracts text directly from memory", func() {
		text, err := ExtractText(ctx, copyOfFakePDF(), "", ExtractionMethodDigital)
		Expect(err).NotTo(HaveOccurred())
		Expect(text).To(ContainSubstring("12345"))
	})

	It("writes nothing to disk in digital mode", func() {
		tmp := tempDirAssertion()

		text, err := ExtractText(ctx, copyOfFakePDF(), "", ExtractionMethodDigital)
		Expect(err).NotTo(HaveOccurred())
		Expect(text).To(ContainSubstring("12345"))

		expectNoTempFiles(tmp)
	})

	It("extracts text via pdftotext without temp files", func() {
		requireTools("pdftotext")
		tmp := tempDirAssertion()

		text, err := ExtractText(ctx, copyOfFakePDF(), "", ExtractionMethodPdftotext)
		Expect(err).NotTo(HaveOccurred())
		Expect(text).To(ContainSubstring("12345"))

		expectNoTempFiles(tmp)
	})

	It("OCRs without temp files", func() {
		requireTools("pdfinfo", "pdftocairo", "tesseract")
		tmp := tempDirAssertion()

		text, err := ExtractText(ctx, copyOfFakePDF(), "", ExtractionMethodOCR)
		Expect(err).NotTo(HaveOccurred())
		Expect(text).To(ContainSubstring("12345"))

		expectNoTempFiles(tmp)
	})

	It("splits tall pages into strips in memory", func() {
		requireTools("pdftocairo", "tesseract")

		pagePNG, err := renderPagePNG(ctx, copyOfFakePDF(), 1)
		Expect(err).NotTo(HaveOccurred())

		img, err := png.Decode(bytes.NewReader(pagePNG))
		Expect(err).NotTo(HaveOccurred())
		bounds := img.Bounds()

		// Stack the page three times: taller than maxStripHeight, so ocrPNG
		// must take the strip path and still terminate.
		tall := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()*3))
		for i := 0; i < 3; i++ {
			draw.Draw(tall, image.Rect(0, bounds.Dy()*i, bounds.Dx(), bounds.Dy()*(i+1)), img, bounds.Min, draw.Src)
		}
		Expect(tall.Bounds().Dy()).To(BeNumerically(">", maxStripHeight))

		var buf bytes.Buffer
		Expect(png.Encode(&buf, tall)).To(Succeed())

		text, err := ocrPNG(ctx, buf.Bytes())
		Expect(err).NotTo(HaveOccurred())
		Expect(text).To(ContainSubstring("12345"))
	})
})

func requireTools(names ...string) {
	for _, name := range names {
		if _, err := exec.LookPath(name); err != nil {
			Skip(name + " not installed")
		}
	}
}
