package service

import (
	"bytes"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"

	"github.com/heyrmi/goslack/util"
	"github.com/stretchr/testify/require"
)

// testFile implements multipart.File interface for testing
type testFile struct {
	*bytes.Reader
}

func (tf *testFile) Close() error {
	return nil
}

func newTestFile(content []byte) multipart.File {
	return &testFile{
		Reader: bytes.NewReader(content),
	}
}

func TestFileService_ValidateFile(t *testing.T) {
	config := util.Config{
		FileMaxSize:      10485760, // 10MB
		FileAllowedTypes: "image/jpeg,image/png,application/pdf,text/plain",
	}

	fileService := &FileService{
		config: config,
	}

	t.Run("ValidFile", func(t *testing.T) {
		header := &multipart.FileHeader{
			Filename: "test.jpg",
			Size:     1024,
			Header:   textproto.MIMEHeader{},
		}
		header.Header.Set("Content-Type", "image/jpeg")

		err := fileService.ValidateFile(header)
		require.NoError(t, err)
	})

	t.Run("FileTooLarge", func(t *testing.T) {
		header := &multipart.FileHeader{
			Filename: "large.jpg",
			Size:     20485760, // 20MB
			Header:   textproto.MIMEHeader{},
		}
		header.Header.Set("Content-Type", "image/jpeg")

		err := fileService.ValidateFile(header)
		require.Error(t, err)
		require.Contains(t, err.Error(), "exceeds maximum allowed size")
	})

	t.Run("InvalidFileType", func(t *testing.T) {
		header := &multipart.FileHeader{
			Filename: "test.exe",
			Size:     1024,
			Header:   textproto.MIMEHeader{},
		}
		header.Header.Set("Content-Type", "application/x-executable")

		err := fileService.ValidateFile(header)
		require.Error(t, err)
		require.Contains(t, err.Error(), "not allowed")
	})

	t.Run("EmptyFile", func(t *testing.T) {
		header := &multipart.FileHeader{
			Filename: "empty.txt",
			Size:     0,
			Header:   textproto.MIMEHeader{},
		}
		header.Header.Set("Content-Type", "text/plain")

		err := fileService.ValidateFile(header)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot be empty")
	})
}

func TestFileService_GenerateUniqueFilename(t *testing.T) {
	fileService := &FileService{}

	t.Run("GeneratesUniqueNames", func(t *testing.T) {
		originalFilename := "test document.pdf"

		filename1 := fileService.GenerateUniqueFilename(originalFilename)
		filename2 := fileService.GenerateUniqueFilename(originalFilename)

		require.NotEqual(t, filename1, filename2)
		require.Contains(t, filename1, "test_document")
		require.Contains(t, filename1, ".pdf")
		require.Contains(t, filename2, "test_document")
		require.Contains(t, filename2, ".pdf")
	})
}

func TestFileService_EnsureUploadDirectory(t *testing.T) {
	tempDir := t.TempDir()
	uploadDir := filepath.Join(tempDir, "test_uploads")

	config := util.Config{
		FileStoragePath: uploadDir,
	}

	fileService := &FileService{
		config: config,
	}

	t.Run("CreatesDirectory", func(t *testing.T) {
		err := fileService.EnsureUploadDirectory()
		require.NoError(t, err)

		// Check that directory was created
		_, err = os.Stat(uploadDir)
		require.NoError(t, err)
	})
}

func TestFileService_CalculateFileHash(t *testing.T) {
	fileService := &FileService{}

	t.Run("CalculatesHash", func(t *testing.T) {
		content := []byte("test file content")
		file := newTestFile(content)

		hash, err := fileService.CalculateFileHash(file)
		require.NoError(t, err)
		require.NotEmpty(t, hash)
		require.Len(t, hash, 64) // SHA-256 produces 64-character hex string

		// Create another file with same content - should get same hash
		file2 := newTestFile(content)
		hash2, err := fileService.CalculateFileHash(file2)
		require.NoError(t, err)
		require.Equal(t, hash, hash2)
	})
}

func TestFileService_GetMimeTypeFromExtension(t *testing.T) {
	fileService := &FileService{}

	testCases := []struct {
		extension string
		expected  string
	}{
		{".jpg", "image/jpeg"},
		{".jpeg", "image/jpeg"},
		{".png", "image/png"},
		{".gif", "image/gif"},
		{".pdf", "application/pdf"},
		{".txt", "text/plain"},
		{".unknown", "application/octet-stream"},
	}

	for _, tc := range testCases {
		t.Run(tc.extension, func(t *testing.T) {
			result := fileService.getMimeTypeFromExtension(tc.extension)
			require.Equal(t, tc.expected, result)
		})
	}
}
