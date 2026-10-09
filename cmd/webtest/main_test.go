package main

import (
    "os"
    "path/filepath"
    "testing"
)
func TestReadSites(t *testing.T) {
    f:=filepath.Join(t.TempDir(),"urls.txt")
    if err:=os.WriteFile(f,[]byte("#comment\nhttps://example.com\nhttps://example.com\nhttps://news.ycombinator.com\n"),0600);err!=nil{t.Fatal(err)}
    sites,err:=readSites(f)
    if err!=nil || len(sites)!=2 {t.Fatalf("sites=%v err=%v",sites,err)}
}
func TestReadSitesInvalid(t *testing.T) {
    f:=filepath.Join(t.TempDir(),"urls.txt")
    _=os.WriteFile(f,[]byte("file:///etc/passwd\n"),0600)
    if _,err:=readSites(f);err==nil{t.Fatal("expected invalid scheme")}
}
func TestLocalSmoke(t *testing.T) {
    srv,url:=defaultSmoke()
    defer srv.Close()
    report:=execute([]string{url},1,"",5000,"local_smoke",nil)
    if report.UsefulPages!=1 {t.Fatalf("report=%+v",report)}
    got:=report.Results[0]
    if got.HTTPStatus!=200 || got.Title!="Local Smoke Test" || got.Links!=1 || got.Forms!=1 {t.Fatalf("bad parsed page: %+v",got)}
}
