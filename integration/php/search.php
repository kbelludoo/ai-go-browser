<?php header('Content-Type: text/html; charset=UTF-8'); $q=htmlspecialchars($_GET['q']??'', ENT_QUOTES, 'UTF-8'); ?>
<!doctype html><title>Pesquisa PHP</title><h1>Pesquisa PHP</h1><p>Consulta: <?= $q ?></p>
