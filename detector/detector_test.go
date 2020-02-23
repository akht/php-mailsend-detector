package detector

import (
	"strings"
	"testing"
)

func TestDetect(t *testing.T) {
	input := `<?php
$to = "mail@example.com";
$subject = "件名";
$body = "本文";
$additional_headers = "追加ヘッダー";
$additional_parameter = "追加パラメタ";
mb_send_mail($to, $subject, $body, $additional_headers, $additional_parameter);`

	expected := `[件名]:
件名
[本文]:
本文`

	src := strings.NewReader(input)
	d := NewDetector(src)
	actual := d.Detect()

	if actual != expected {
		t.Fatalf("fail. expected=%q, got=%q", expected, actual)
	}
}
