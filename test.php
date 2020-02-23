<?php

$to = "mail@example.com";
$subject = "件名";
$body = "本文";
$additional_headers = "追加ヘッダー";
$additional_parameter = "追加パラメタ";

mb_send_mail($to, $subject, $body, $additional_headers, $additional_parameter);
