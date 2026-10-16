<?php
header('Content-Type: application/json');
if (($_COOKIE['session'] ?? '') !== 'php-session') {http_response_code(401); echo json_encode(['error'=>'no cookie']); exit;}
echo json_encode(['backend' => 'php', 'products' => [['name' => 'Caneta'], ['name' => 'Livro']], 'api_token' => 'secret-value']);
