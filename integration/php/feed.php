<?php
header('Content-Type: application/json');
echo json_encode(['backend' => 'php', 'feed' => ['item' => 123]]);
