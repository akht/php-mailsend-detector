package detector

import (
	"strings"
	"testing"
)

func TestDetect(t *testing.T) {
	tests := []string{
		// 一番素朴なパターン
		`<?php
$to = "mail@example.com";
$subject = "件名";
$body = "本文";
$additional_headers = "追加ヘッダー";
$additional_parameter = "追加パラメタ";
mb_send_mail($to, $subject, $body, $additional_headers, $additional_parameter);`,

		// 変数に定数を代入
		`<?php
$CONSTANT = "本文";
$to = "mail@example.com";
$subject = "件名";
$body = $CONSTANT;
$additional_headers = "追加ヘッダー";
$additional_parameter = "追加パラメタ";
mb_send_mail($to, $subject, $body, $additional_headers, $additional_parameter);`,

		// 関数の戻り値を代入
		`<?php

$CONSTANT = "本文";

$to = "mail@example.com";
$subject = get_subject();
$body = $CONSTANT;
$additional_headers = "追加ヘッダー";
$additional_parameter = "追加パラメタ";

function get_subject() {
	return "件名";
}

mb_send_mail($to, $subject, $body, $additional_headers, $additional_parameter);`,
	}

	expected := `[件名]:
件名
[本文]:
本文`

	for _, input := range tests {
		src := strings.NewReader(input)
		d := NewDetector(src)
		actual := d.Detect()

		if actual != expected {
			t.Fatalf("fail. expected=%q, got=%q", expected, actual)
		}
	}
}
