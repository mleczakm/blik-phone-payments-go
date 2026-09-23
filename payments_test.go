package payments

import "testing"

func TestParseAmount(t *testing.T) {
	cases := map[string]int64{"123,45": 12345, "1 234,56": 123456, "1234.5": 123450, "12": 1200, "0,07": 7, "258,99": 25899}
	for in, want := range cases {
		got, err := ParseAmount(in)
		if err != nil || got != want {
			t.Errorf("ParseAmount(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "abc", "1,234,5", "1.999", "-1", "1.-1", "92233720368547759"} {
		if _, err := ParseAmount(bad); err == nil {
			t.Errorf("ParseAmount(%q) should fail", bad)
		}
	}
}

func TestAliorParser(t *testing.T) {
	subject := "Uznanie rachunku 12...3456 kwotą 258,99 PLN"
	body := `<p>Uznanie rachunku kwotą 258,99 PLN<br>Nadawca: ANNA TESTOWA<br>Tytuł zlecenia: AKWU zamówienie<br></p>`
	n, ok := AliorParser{}.Parse(subject, body)
	if !ok || n.Amount != 25899 || n.Sender != "ANNA TESTOWA" || n.Title != "AKWU zamówienie" {
		t.Fatalf("unexpected %+v %v", n, ok)
	}
	if _, ok := (AliorParser{}).Parse("Obciążenie rachunku", body); ok {
		t.Fatal("debit must not parse")
	}
}

func TestGenericParser(t *testing.T) {
	n, ok := GenericParser{}.Parse("Uznanie rachunku - Kwota: 1 234,56 PLN - Nadawca: Jan Kowalski", "Tytuł: zakup K7PX\n")
	if !ok || n.Amount != 123456 || n.Sender != "Jan Kowalski" || n.Title != "zakup K7PX" {
		t.Fatalf("unexpected %+v", n)
	}
}

func TestFindCode(t *testing.T) {
	codes := []string{"AKWU", "10ZX", "B7Q2"}
	cases := map[string]string{
		"AKWU":                   "AKWU",
		"zamówienie akwu dzięki": "AKWU",
		"IOZX":                   "10ZX", // I→1, O→0 typos
		"b7q2.":                  "B7Q2",
		"AKWUX":                  "",
		"brak kodu":              "",
	}
	for title, want := range cases {
		if got := FindCode(title, codes); got != want {
			t.Errorf("FindCode(%q) = %q, want %q", title, got, want)
		}
	}
}

func TestGenerateCode(t *testing.T) {
	for range 100 {
		c, err := GenerateCode()
		if err != nil || len(c) != CodeLength {
			t.Fatalf("bad code %q %v", c, err)
		}
		if !IsValidCode(c) {
			t.Fatalf("generated invalid code %q", c)
		}
		for _, r := range c {
			if r == 'I' || r == 'O' {
				t.Fatalf("ambiguous char in %q", c)
			}
		}
	}
}

func TestIsValidCode(t *testing.T) {
	for _, code := range []string{"AB12", "0000", "abcd"} {
		if !IsValidCode(code) {
			t.Errorf("IsValidCode(%q) = false", code)
		}
	}
	for _, code := range []string{"AIO1", "ABC", "ABCDE", "ĄBCD", "１２３４"} {
		if IsValidCode(code) {
			t.Errorf("IsValidCode(%q) = true", code)
		}
	}
}
