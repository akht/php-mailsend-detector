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
		define("CONSTANT", "This is a Body");
		$to = "mail@example.com";
		$subject = "This is a Subject";
		$body = CONSTANT;
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
			$ret = "This is a Subject";
			return $ret;
		}

		mb_send_mail($to, $subject, $body, $additional_headers, $additional_parameter);`,

		// 関数の戻り値に定数を使用
		`<?php
		define("CONSTANT_SUBJECT", "This is a Subject");
		define("CONSTANT_BODY", "This is a Body");

		$to = "mail@example.com";
		$subject = get_subject();
		$body = CONSTANT_BODY;
		$additional_headers = "追加ヘッダー";
		$additional_parameter = "追加パラメタ";

		function get_subject() {
			return CONSTANT_SUBJECT;
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

		// 文字列の結合、定数との結合、関数内のローカル変数を参照して返すなど
		`<?php
		define("CONSTANT_SUBJECT1", "This ");
		define("CONSTANT_SUBJECT2", "is ");
		define("CONSTANT_BODY1", "Thi");
		define("CONSTANT_BODY2", "a");

		$to = "mail@example.com";
		$subject = get_subject() . "a Subject";
		$body = CONSTANT_BODY1 . get_body1() . get_body2() . " " . get_body3();
		$additional_headers = "追加ヘッダー";
		$additional_parameter = "追加パラメタ";

		function get_subject() {
			return CONSTANT_SUBJECT1 . CONSTANT_SUBJECT2;
		}

		function get_body1() {
		    return "s ";
		}

		function get_body2() {
			$var = "i";
			$var2 = $var . get_s();
		    return $var2;
		}

		function get_body3() {
		    $to3 = CONSTANT_BODY2 . " " . "Body";
		    return $to3;
		}

		function get_s() {
			return "s";
		}

		mb_send_mail($to, $subject, $body, $additional_headers, $additional_parameter);`,

		// 同じ変数に複数回代入
		`<?php
		$to = "mail@example.com";
		$subject = "This is a ";
		$subject = $subject . "Subject";
		$body1 = "This";
		$body2 = "is";
		$body3 = "a";
		$body = $body1 . " " . $body2 . " " . $body3 . " " . "Body";
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
