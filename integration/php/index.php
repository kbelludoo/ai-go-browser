<?php
setcookie('session', 'php-session', ['path' => '/', 'httponly' => true]);
header('Content-Type: text/html; charset=UTF-8');
?>
<!doctype html><html lang="pt-BR"><head><title>PHP real</title>
<link rel="alternate" type="application/json" title="PHP JSON feed" href="/feed.php"></head><body>
<h1>PHP real</h1>
<button id="products" onclick="fetch('/products.php')">Ver produtos PHP</button>
<a href="/article.php?topic=go">Ler artigo PHP</a>
<form action="/search.php" method="GET"><input type="text" name="q" value="antes"><button>Pesquisar PHP</button></form>
<button hx-get="/fragment.php">Carregar fragmento PHP</button>
<form action="/danger.php" method="POST"><input name="danger" value="no"><button>Enviar perigoso</button></form>
<a href="http://localhost:12345/forbidden">Link fora da allowlist</a>
</body></html>
