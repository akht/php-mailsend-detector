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
$subject = "This is a Subject";
$body = "This is a Body";
$additional_headers = "追加ヘッダー";
$additional_parameter = "追加パラメタ";
mb_send_mail($to, $subject, $body, $additional_headers, $additional_parameter);`,

		// 変数に定数を代入
		`<?php
$CONSTANT = "This is a Body";
$to = "mail@example.com";
$subject = "This is a Subject";
$body = $CONSTANT;
$additional_headers = "追加ヘッダー";
$additional_parameter = "追加パラメタ";
mb_send_mail($to, $subject, $body, $additional_headers, $additional_parameter);`,

		// 関数の戻り値を代入
		`<?php

$CONSTANT = "This is a Body";

$to = "mail@example.com";
$subject = get_subject();
$body = $CONSTANT;
$additional_headers = "追加ヘッダー";
$additional_parameter = "追加パラメタ";

function get_subject() {
	return "This is a Subject";
}

mb_send_mail($to, $subject, $body, $additional_headers, $additional_parameter);`,

		// 関数の戻り値に定数を使用
		`<?php
$CONSTANT_SUBJECT = "This is a Subject";
$CONSTANT_BODY = "This is a Body";

$to = "mail@example.com";
$subject = get_subject();
$body = $CONSTANT_BODY;
$additional_headers = "追加ヘッダー";
$additional_parameter = "追加パラメタ";

function get_subject() {
	return $CONSTANT_SUBJECT;
}

mb_send_mail($to, $subject, $body, $additional_headers, $additional_parameter);`,

		// 文字列結合
		`<?php
$to = "mail@example.com";
$subject = "This is" . " a Subject";
$body = "This is a Body" . "";
$additional_headers = "追加ヘッダー";
$additional_parameter = "追加パラメタ";
mb_send_mail($to, $subject, $body, $additional_headers, $additional_parameter);`,
	}

	expected := `[件名]:
This is a Subject
[本文]:
This is a Body`

	for _, input := range tests {
		src := strings.NewReader(input)
		d := NewDetector(src)
		actual := d.Detect()

		if actual != expected {
			t.Fatalf("fail. expected=%q, got=%q", expected, actual)
		}
	}
}
