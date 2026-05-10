package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

func utf8BOM(s string) []byte {
	return append([]byte{0xEF, 0xBB, 0xBF}, []byte(s)...)
}

func toShiftJIS(s string) []byte {
	out, _, err := transform.Bytes(japanese.ShiftJIS.NewEncoder(), []byte(s))
	if err != nil {
		panic(err)
	}
	return out
}

func serveCSV(t *testing.T, body []byte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
}

// --- escapeText ---

func TestEscapeText(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"empty", "", ""},
		{"no special chars", "元日", "元日"},
		{"semicolon", "a;b", `a\;b`},
		{"comma", "a,b", `a\,b`},
		{"LF", "a\nb", `a\nb`},
		{"CR removed", "a\rb", "ab"},
		{"CRLF", "a\r\nb", `a\nb`},
		{"single backslash", `a\b`, `a\\b`},
		{"double backslash", `a\\b`, `a\\\\b`},
		{"multiple specials", "a;b,c\nd", `a\;b\,c\nd`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeText(tt.in)
			if got != tt.want {
				t.Errorf("escapeText(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// --- buildUID ---

func TestBuildUID_Deterministic(t *testing.T) {
	h := Holiday{Date: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Title: "元日"}
	const want = "8dd2601840f5c6f535662d7e3847f557@jp-holidays-cao-example"
	if got := buildUID(h); got != want {
		t.Errorf("buildUID = %q, want %q", got, want)
	}
}

func TestBuildUID_DifferentHolidaysProduceDifferentUIDs(t *testing.T) {
	h1 := Holiday{Date: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Title: "元日"}
	h2 := Holiday{Date: time.Date(2026, 1, 13, 0, 0, 0, 0, time.UTC), Title: "成人の日"}
	if buildUID(h1) == buildUID(h2) {
		t.Error("expected different UIDs for different holidays")
	}
}

func TestBuildUID_SameDateDifferentTitleProducesDifferentUID(t *testing.T) {
	base := time.Date(2032, 11, 3, 0, 0, 0, 0, time.UTC)
	h1 := Holiday{Date: base, Title: "文化の日"}
	h2 := Holiday{Date: base, Title: "休日"}
	if buildUID(h1) == buildUID(h2) {
		t.Error("same date different title should produce different UIDs")
	}
}

// --- buildICS ---

func TestBuildICS_RequiredStructure(t *testing.T) {
	ics, err := buildICS([]Holiday{
		{Date: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Title: "元日"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body := string(ics)
	for _, want := range []string{
		"BEGIN:VCALENDAR", "END:VCALENDAR",
		"BEGIN:VEVENT", "END:VEVENT",
		"VERSION:2.0", "CALSCALE:GREGORIAN",
		"X-WR-TIMEZONE:Asia/Tokyo",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("ICS missing required field: %q", want)
		}
	}
}

func TestBuildICS_AllDayExclusiveEnd(t *testing.T) {
	// 終日イベントの DTEND は翌日（exclusive）
	ics, err := buildICS([]Holiday{
		{Date: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Title: "元日"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body := string(ics)
	if !strings.Contains(body, "DTSTART;VALUE=DATE:20260101") {
		t.Error("expected DTSTART;VALUE=DATE:20260101")
	}
	if !strings.Contains(body, "DTEND;VALUE=DATE:20260102") {
		t.Error("expected DTEND;VALUE=DATE:20260102 (exclusive end)")
	}
}

func TestBuildICS_YearEndBoundary(t *testing.T) {
	// 12/31 の DTEND は翌年 1/1
	ics, err := buildICS([]Holiday{
		{Date: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC), Title: "大晦日"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(ics), "DTEND;VALUE=DATE:20260101") {
		t.Error("expected DTEND;VALUE=DATE:20260101 across year boundary")
	}
}

func TestBuildICS_LeapYearFeb29(t *testing.T) {
	// 2028年は閏年: 2/29 の翌日は 3/1
	ics, err := buildICS([]Holiday{
		{Date: time.Date(2028, 2, 29, 0, 0, 0, 0, time.UTC), Title: "テスト"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body := string(ics)
	if !strings.Contains(body, "DTSTART;VALUE=DATE:20280229") {
		t.Error("expected DTSTART;VALUE=DATE:20280229")
	}
	if !strings.Contains(body, "DTEND;VALUE=DATE:20280301") {
		t.Error("expected DTEND;VALUE=DATE:20280301")
	}
}

func TestBuildICS_MonthEndBoundary(t *testing.T) {
	// 1/31 の翌日は 2/1
	ics, err := buildICS([]Holiday{
		{Date: time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), Title: "テスト"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(ics), "DTEND;VALUE=DATE:20260201") {
		t.Error("expected DTEND;VALUE=DATE:20260201 across month boundary")
	}
}

func TestBuildICS_UsesCRLF(t *testing.T) {
	ics, err := buildICS([]Holiday{
		{Date: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Title: "元日"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains(ics, []byte("\r\n")) {
		t.Error("ICS must use CRLF line endings (RFC 5545)")
	}
}

// --- fetchHolidays ---

func TestFetchHolidays_ShiftJIS(t *testing.T) {
	body := toShiftJIS("月日,祝日名称\r\n2026/1/1,元日\r\n2026/1/13,成人の日\r\n")
	srv := serveCSV(t, body)
	defer srv.Close()

	holidays, err := fetchHolidays(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(holidays) != 2 {
		t.Fatalf("expected 2 holidays, got %d", len(holidays))
	}
	if holidays[0].Title != "元日" {
		t.Errorf("expected 元日, got %q", holidays[0].Title)
	}
}

func TestFetchHolidays_UTF8BOM(t *testing.T) {
	body := utf8BOM("月日,祝日名称\r\n2026/1/1,元日\r\n")
	srv := serveCSV(t, body)
	defer srv.Close()

	holidays, err := fetchHolidays(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(holidays) != 1 || holidays[0].Title != "元日" {
		t.Errorf("expected [元日], got %v", holidays)
	}
}

func TestFetchHolidays_SortedByDate(t *testing.T) {
	// CSV が逆順でも日付昇順で返ること
	body := utf8BOM("月日,祝日名称\r\n2026/1/13,成人の日\r\n2026/1/1,元日\r\n")
	srv := serveCSV(t, body)
	defer srv.Close()

	holidays, err := fetchHolidays(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(holidays) != 2 {
		t.Fatalf("expected 2, got %d", len(holidays))
	}
	if holidays[0].Title != "元日" {
		t.Errorf("expected 元日 first (Jan 1), got %q", holidays[0].Title)
	}
}

func TestFetchHolidays_SameDateSortedByTitle(t *testing.T) {
	// 同日エントリは Unicode 昇順（休日 U+4F11 < 文化の日 U+6587）
	body := utf8BOM("月日,祝日名称\r\n2032/11/3,文化の日\r\n2032/11/3,休日\r\n")
	srv := serveCSV(t, body)
	defer srv.Close()

	holidays, err := fetchHolidays(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(holidays) != 2 {
		t.Fatalf("expected 2, got %d", len(holidays))
	}
	if holidays[0].Title != "休日" {
		t.Errorf("expected 休日 before 文化の日, got %q", holidays[0].Title)
	}
}

func TestFetchHolidays_EmptyCSV(t *testing.T) {
	srv := serveCSV(t, []byte{})
	defer srv.Close()

	_, err := fetchHolidays(context.Background(), srv.URL)
	if err == nil {
		t.Error("expected error for empty CSV")
	}
}

func TestFetchHolidays_HeaderOnly(t *testing.T) {
	// ヘッダ行のみ → エラーなし、0件
	body := utf8BOM("月日,祝日名称\r\n")
	srv := serveCSV(t, body)
	defer srv.Close()

	holidays, err := fetchHolidays(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(holidays) != 0 {
		t.Errorf("expected 0 holidays for header-only CSV, got %d", len(holidays))
	}
}

func TestFetchHolidays_SkipsRowsWithTooFewFields(t *testing.T) {
	// フィールドが1列の行はスキップ
	body := utf8BOM("月日,祝日名称\r\n2026/1/1\r\n2026/1/13,成人の日\r\n")
	srv := serveCSV(t, body)
	defer srv.Close()

	holidays, err := fetchHolidays(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(holidays) != 1 {
		t.Errorf("expected 1 holiday (1-field row skipped), got %d", len(holidays))
	}
}

func TestFetchHolidays_SkipsEmptyFields(t *testing.T) {
	// date または title が空の行はスキップ
	body := utf8BOM("月日,祝日名称\r\n,\r\n2026/1/1,元日\r\n2026/1/13,\r\n")
	srv := serveCSV(t, body)
	defer srv.Close()

	holidays, err := fetchHolidays(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(holidays) != 1 {
		t.Errorf("expected 1 holiday (empty-field rows skipped), got %d", len(holidays))
	}
}

func TestFetchHolidays_WhitespaceTrimmed(t *testing.T) {
	// フィールドの前後空白はトリムされる
	body := utf8BOM("月日,祝日名称\r\n 2026/1/1 , 元日 \r\n")
	srv := serveCSV(t, body)
	defer srv.Close()

	holidays, err := fetchHolidays(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(holidays) != 1 {
		t.Fatalf("expected 1 holiday, got %d", len(holidays))
	}
	if holidays[0].Title != "元日" {
		t.Errorf("expected trimmed title 元日, got %q", holidays[0].Title)
	}
}

func TestFetchHolidays_InvalidDateFormat(t *testing.T) {
	// ハイフン区切り日付はエラー（内閣府形式はスラッシュ区切り）
	srv := serveCSV(t, []byte("2026-01-01,Holiday\r\n"))
	defer srv.Close()

	_, err := fetchHolidays(context.Background(), srv.URL)
	if err == nil {
		t.Error("expected error for hyphen-separated date format")
	}
}

func TestFetchHolidays_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := fetchHolidays(context.Background(), srv.URL)
	if err == nil {
		t.Error("expected error for HTTP 500")
	}
}

func TestFetchHolidays_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := fetchHolidays(ctx, srv.URL)
	if err == nil {
		t.Error("expected error when context is cancelled")
	}
}

// --- writeIfChanged ---

func TestWriteIfChanged_CreatesNewFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.ics")
	data := []byte("BEGIN:VCALENDAR\r\nEND:VCALENDAR\r\n")

	changed, err := writeIfChanged(path, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Error("expected changed=true for new file")
	}
	got, _ := os.ReadFile(path)
	if !bytes.Equal(got, data) {
		t.Error("file content mismatch after create")
	}
}

func TestWriteIfChanged_IdenticalContentNoWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.ics")
	data := []byte("BEGIN:VCALENDAR\r\nEND:VCALENDAR\r\n")
	os.WriteFile(path, data, 0o644)

	changed, err := writeIfChanged(path, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed {
		t.Error("expected changed=false for identical content")
	}
}

func TestWriteIfChanged_DifferentContentOverwrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.ics")
	os.WriteFile(path, []byte("old content"), 0o644)
	newData := []byte("new content")

	changed, err := writeIfChanged(path, newData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Error("expected changed=true when content differs")
	}
	got, _ := os.ReadFile(path)
	if !bytes.Equal(got, newData) {
		t.Error("file should contain updated content")
	}
}

func TestWriteIfChanged_EmptyFileGetsContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.ics")
	os.WriteFile(path, []byte{}, 0o644)
	data := []byte("content")

	changed, err := writeIfChanged(path, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Error("expected changed=true when writing to empty file")
	}
}

func TestWriteIfChanged_ContentToEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.ics")
	os.WriteFile(path, []byte("content"), 0o644)

	changed, err := writeIfChanged(path, []byte{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Error("expected changed=true when truncating to empty")
	}
}
