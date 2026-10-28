package services

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rupee-os/rupee-os/internal/parser"
)

func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func TransactionDedupHash(t parser.ExtractedTransaction) string {
	key := fmt.Sprintf("%s|%s|%d|%s",
		t.Date.UTC().Format("2006-01-02"),
		strings.TrimSpace(strings.ToLower(t.Description)),
		t.AmountPaise,
		t.Direction,
	)
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func MonthStart(t time.Time) time.Time {
	y, m, _ := t.Date()
	return time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
}
