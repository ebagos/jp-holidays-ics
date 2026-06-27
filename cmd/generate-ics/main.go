package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

const (
	sourceURL     = "https://www8.cao.go.jp/chosei/shukujitsu/syukujitsu.csv"
	outputPath    = "public/japanese-holidays.ics"
	calendarID    = "jp-holidays-cao-example"
	calendarName  = "日本の祝日"
	maxLineOctets = 75
)

type Holiday struct {
	Date  time.Time
	Title string
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	holidays, err := fetchHolidays(ctx, sourceURL)
	if err != nil {
		exitf("fetch failed: %v", err)
	}

	if len(holidays) == 0 {
		exitf("no holidays found")
	}

	ics, err := buildICS(holidays)
	if err != nil {
		exitf("build ics failed: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		exitf("mkdir failed: %v", err)
	}

	changed, err := writeIfChanged(outputPath, ics)
	if err != nil {
		exitf("write failed: %v", err)
	}

	if changed {
		fmt.Printf("updated: %s\n", outputPath)
	} else {
		fmt.Printf("no change: %s\n", outputPath)
	}
}

func fetchHolidays(ctx context.Context, url string) ([]Holiday, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// UTF-8 BOM除去、またはShift-JISをUTF-8に変換
	if bytes.HasPrefix(body, []byte{0xEF, 0xBB, 0xBF}) {
		body = body[3:]
	} else {
		decoded, err := io.ReadAll(transform.NewReader(bytes.NewReader(body), japanese.ShiftJIS.NewDecoder()))
		if err != nil {
			return nil, fmt.Errorf("decode shift-jis: %w", err)
		}
		body = decoded
	}

	r := csv.NewReader(bytes.NewReader(body))
	r.FieldsPerRecord = -1

	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, errors.New("empty csv")
	}

	var holidays []Holiday

	for i, row := range rows {
		if len(row) < 2 {
			continue
		}

		dateStr := strings.TrimSpace(row[0])
		title := strings.TrimSpace(row[1])

		// ヘッダ行スキップ
		if i == 0 && (strings.Contains(dateStr, "月日") || strings.Contains(title, "名称")) {
			continue
		}
		if dateStr == "" || title == "" {
			continue
		}

		// 内閣府CSV形式: 2026/1/1 (ゼロ埋めなし)
		d, err := time.Parse("2006/1/2", dateStr)
		if err != nil {
			// 想定外フォーマットは無視せず落としたい場合は return err
			return nil, fmt.Errorf("invalid date %q: %w", dateStr, err)
		}

		holidays = append(holidays, Holiday{
			Date:  d,
			Title: title,
		})
	}

	sort.Slice(holidays, func(i, j int) bool {
		if holidays[i].Date.Equal(holidays[j].Date) {
			return holidays[i].Title < holidays[j].Title
		}
		return holidays[i].Date.Before(holidays[j].Date)
	})

	return holidays, nil
}

func buildICS(holidays []Holiday) ([]byte, error) {
	var b strings.Builder

	writeLine(&b, "BEGIN:VCALENDAR")
	writeLine(&b, "PRODID:-//example//Japan Holidays from CAO//JA")
	writeLine(&b, "VERSION:2.0")
	writeLine(&b, "CALSCALE:GREGORIAN")
	writeLine(&b, "METHOD:PUBLISH")
	writeLine(&b, "X-WR-CALNAME:"+escapeText(calendarName))
	writeLine(&b, "X-WR-CALDESC:"+escapeText("内閣府 公開CSVから自動生成した日本の祝日カレンダー"))
	writeLine(&b, "X-WR-TIMEZONE:Asia/Tokyo")

	for _, h := range holidays {
		start := h.Date.Format("20060102")
		end := h.Date.AddDate(0, 0, 1).Format("20060102") // 終日は翌日exclusive
		uid := buildUID(h)

		writeLine(&b, "BEGIN:VEVENT")
		writeLine(&b, "UID:"+uid)
		writeLine(&b, "DTSTAMP:"+buildDTStamp(h))
		writeLine(&b, "DTSTART;VALUE=DATE:"+start)
		writeLine(&b, "DTEND;VALUE=DATE:"+end)
		writeLine(&b, "SUMMARY:"+escapeText(h.Title))
		writeLine(&b, "DESCRIPTION:"+escapeText("Source: Cabinet Office, Government of Japan"))
		writeLine(&b, "TRANSP:TRANSPARENT")
		writeLine(&b, "STATUS:CONFIRMED")
		writeLine(&b, "END:VEVENT")
	}

	writeLine(&b, "END:VCALENDAR")

	return []byte(b.String()), nil
}

func buildUID(h Holiday) string {
	sum := sha256.Sum256([]byte(h.Date.Format("2006-01-02") + "|" + h.Title + "|" + calendarID))
	return fmt.Sprintf("%x@%s", sum[:16], calendarID)
}

func buildDTStamp(h Holiday) string {
	return h.Date.Format("20060102") + "T000000Z"
}

func writeLine(b *strings.Builder, line string) {
	for i, folded := range foldLine(line) {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(folded)
		b.WriteString("\r\n")
	}
}

func foldLine(line string) []string {
	if len([]byte(line)) <= maxLineOctets {
		return []string{line}
	}

	lines := make([]string, 0, len(line)/maxLineOctets+1)
	var current strings.Builder
	currentOctets := 0
	limit := maxLineOctets

	for _, r := range line {
		runeOctets := len(string(r))
		if currentOctets > 0 && currentOctets+runeOctets > limit {
			lines = append(lines, current.String())
			current.Reset()
			currentOctets = 0
			limit = maxLineOctets - 1
		}
		current.WriteRune(r)
		currentOctets += runeOctets
	}

	if currentOctets > 0 {
		lines = append(lines, current.String())
	}

	return lines
}

func escapeText(s string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`;`, `\;`,
		`,`, `\,`,
		"\n", `\n`,
		"\r", ``,
	)
	return replacer.Replace(s)
}

func writeIfChanged(path string, data []byte) (bool, error) {
	old, err := os.ReadFile(path)
	if err == nil && bytes.Equal(old, data) {
		return false, nil
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return false, err
	}
	return true, nil
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
