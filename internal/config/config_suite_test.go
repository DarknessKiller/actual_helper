package config_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// writeTempConfig writes a config document to a temp file and returns its
// path. Test data is fully fake.
func writeTempConfig(content string) string {
	dir := GinkgoT().TempDir()
	path := filepath.Join(dir, "provider_config.json")
	Expect(os.WriteFile(path, []byte(content), 0644)).To(Succeed())
	return path
}

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Suite")
}
