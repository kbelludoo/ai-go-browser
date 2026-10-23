$spKey = "$env:TEMP\key_ssh-key-2026-10-09 sao paulo.key.key"
$pubKey = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDFuMXGbJm6U99H8SEhdItQ1n/2DbtVCMyZsXIR6v+xPMYfqS45wvQaHz9w9SAcaC88dHf02LJe9omsu3oqViZKurt51GZVsJYBrsLNZttGyu8qrEIPPKFoajW2EOl+9iwUfxh0O4s8MypzeGvFhhlIymiG2EFaFXSQBAD4lvVLtAMYYPDjlQAFSDv8qeXw/A1WWIFpBqaGSXdJXk296MvYpjs17L9uhn0uc1r4KPazNzgJIw+p4hb9rkO7t706kOTBNfXoznfKqGsMbucId9hqDJJHcO265Rp7HX3HO/LA5Kh23+HMVZD8EzxQmznl2WEwyqC+r8cggu1Bm+tNF63D"

$cmd = "echo '$pubKey' >> ~/.ssh/authorized_keys; sort -u ~/.ssh/authorized_keys -o ~/.ssh/authorized_keys; echo KEY_ADDED_SUCCESS"
ssh -i $spKey -o StrictHostKeyChecking=no ubuntu@147.15.109.14 $cmd
