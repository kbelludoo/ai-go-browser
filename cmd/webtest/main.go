// webtest tests the AI Go Browser's ability to extract useful content from
// explicit, read-only websites. It does NOT bypass bot defenses.
package main

import (
    "ai-go-browser/browser"
    "bufio"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "net/http"
    "net/http/httptest"
    "net/url"
    "os"
    "path/filepath"
    "strings"
    "time"
)

type Site struct {
    URL string `json:"url"`
    Run int `json:"run"`
    Loaded bool `json:"loaded"`
    UsefulText bool `json:"useful_text"`
    HTTPStatus int `json:"http_status,omitempty"`
    Title string `json:"title,omitempty"`
    TextPreview string `json:"text_preview,omitempty"`
    Links int `json:"links"`
    Forms int `json:"forms"`
    Buttons int `json:"buttons"`
    Headings int `json:"headings"`
    MarkdownBytes int `json:"markdown_bytes"`
    BodyBytes int `json:"body_bytes"`
    DurationMs int64 `json:"duration_ms"`
    GoRSSMiB float64 `json:"go_rss_mib"`
    WorkerPeakRSSMiB float64 `json:"js_worker_peak_rss_mib,omitempty"`
    ApproxRSSMiB float64 `json:"approx_sum_rss_mib,omitempty"`
    JSAttempted int `json:"js_attempted,omitempty"`
    JSExecuted int `json:"js_executed,omitempty"`
    JSErrors []string `json:"js_errors,omitempty"`
    Error string `json:"error,omitempty"`
}

type Report struct {
    Version string `json:"version"`
    MeasuredAt string `json:"measured_at_utc"`
    Mode string `json:"mode"`
    Engine string `json:"engine"`
    JavaScript bool `json:"javascript_enabled"`
    Repeat int `json:"repeat"`
    SiteCount int `json:"site_count"`
    SuccessfulLoads int `json:"successful_loads"`
    UsefulPages int `json:"useful_pages"`
    MaxObservedGoRSSMiB float64 `json:"max_observed_go_rss_mib"`
    MaxObservedApproxRSSMiB float64 `json:"max_observed_approx_rss_mib"`
    Notes []string `json:"notes"`
    Results []Site `json:"results"`
}

func readSites(path string) ([]string,error) {
    f,err:=os.Open(path); if err!=nil {return nil,err}; defer f.Close()
    s:=bufio.NewScanner(f)
    var out []string
    seen:=map[string]bool{}
    for s.Scan() {
        line:=strings.TrimSpace(s.Text())
        if line=="" || strings.HasPrefix(line,"#") {continue}
        u,err:=url.Parse(line)
        if err!=nil || u.Hostname()=="" || (u.Scheme!="http" && u.Scheme!="https") || u.User!=nil {
            return nil,fmt.Errorf("invalid site %q",line)
        }
        if !seen[line] {seen[line]=true;out=append(out,line)}
        if len(out)>20 {return nil,errors.New("limit: 20 sites per run")}
    }
    if err:=s.Err();err!=nil{return nil,err}
    if len(out)==0{return nil,errors.New("no URLs in sites file")}
    return out,nil
}
func defaultSmoke() (*httptest.Server,string) {
    mux:=http.NewServeMux()
    mux.HandleFunc("/",func(w http.ResponseWriter,r *http.Request){
        w.Header().Set("Content-Type","text/html; charset=utf-8")
        fmt.Fprint(w,`<!doctype html><html><head><title>Local Smoke Test</title></head><body><main><h1>Sample article</h1><p id="message">Readable article text for our AI agent.</p><a href="/next">Next page</a><form method="GET" action="/search"><input id="q" name="q"></form><script>document.getElementById('message').textContent='JavaScript changed the article'</script></main></body></html>`)
    })
    mux.HandleFunc("/next",func(w http.ResponseWriter,r *http.Request){fmt.Fprint(w,"<title>Next</title><h1>Next page</h1>")})
    srv:=httptest.NewServer(mux)
    return srv,srv.URL
}
func probe(raw string,run int,worker string,maxText int,extraHosts []string) Site {
    b:=browser.New([]string{"*"},false)
    b.AllowAll=true
    b.JSWorker=worker
    b.MaxText=maxText
    defer b.Close()
    start:=time.Now()
    result:=b.Execute(browser.Command{Cmd:"open",URL:raw})
    s:=Site{URL:raw,Run:run,DurationMs:time.Since(start).Milliseconds(),GoRSSMiB:result.Memory.RSSMiB}
    if !result.OK {s.Error=result.Error;return s}
    if result.Page==nil {s.Error="no page returned";return s}
    p:=result.Page
    s.HTTPStatus=p.Status
    s.Title=p.Title
    s.BodyBytes=p.Bytes
    s.Links=len(p.Links)
    s.Forms=len(p.Forms)
    s.Buttons=len(p.Buttons)
    s.Headings=len(p.Headings)
    s.MarkdownBytes=len(p.Markdown)
    t:=strings.TrimSpace(p.Text)
    if len(t)>200 {t=t[:200]}
    s.TextPreview=t
    s.Loaded = p.Status>=200 && p.Status<300
    // A 200 OK alone is not evidence that the agent can understand the page.
    s.UsefulText = s.Loaded && len(strings.TrimSpace(p.Text)) >= 40
    if result.JS != nil {
        s.JSAttempted=result.JS.Attempted
        s.JSExecuted=result.JS.Executed
        s.JSErrors=result.JS.Errors
        s.WorkerPeakRSSMiB=result.JS.WorkerPeakRSSMiB
    }
    // Diagnostic only: different sampling times, not a synchronized memory peak.
    s.ApproxRSSMiB=s.GoRSSMiB+s.WorkerPeakRSSMiB
    return s
}
func execute(urls []string,repeats int,worker string,maxText int,mode string,extraHosts []string) Report {
    r:=Report{
        Version:"0.6-webtest",MeasuredAt:time.Now().UTC().Format(time.RFC3339),
        Mode:mode,Engine:"Go HTTP+incremental DOM; optional Duktape worker",JavaScript:worker!="",
        Repeat:repeats,SiteCount:len(urls),
        Notes: []string{
            "Results are from the environment where webtest is actually run; local_smoke is not a public-website test.",
            "A valid HTTP response does not establish JavaScript/DOM parity with Chromium.",
            "Go RSS is sampled after each command, not continuously; true peak may be higher.",
            "Approx RSS sums Go RSS and worker historical RSS peaks measured at different instants; it is not a process-tree peak or a 300-MiB guarantee.",
            "Only GET requests are issued. No login, form submission, CAPTCHA circumvention or anti-bot bypass.",
        },
    }
    for _,site:=range urls {
        for i:=1;i<=repeats;i++ {
            result:=probe(site,i,worker,maxText,extraHosts)
            if result.Loaded {r.SuccessfulLoads++}
            if result.UsefulText {r.UsefulPages++}
            if result.GoRSSMiB>r.MaxObservedGoRSSMiB{r.MaxObservedGoRSSMiB=result.GoRSSMiB}
            if result.ApproxRSSMiB>r.MaxObservedApproxRSSMiB{r.MaxObservedApproxRSSMiB=result.ApproxRSSMiB}
            r.Results=append(r.Results,result)
            fmt.Fprintf(os.Stderr,"%-41s HTTP=%3d title=%q text=%t rss=%.1fMiB err=%s\n",site,result.HTTPStatus,result.Title,result.UsefulText,result.GoRSSMiB,result.Error)
        }
    }
    return r
}
func main() {
    sites:=flag.String("sites","webtest/sites.txt","list of URL(s), one per line")
    output:=flag.String("out","webtest/report.json","JSON report destination")
    worker:=flag.String("js-worker","","path to optional Duktape worker (match the host architecture)")
    repeat:=flag.Int("repeat",1,"number of runs per URL (1..10)")
    maxText:=flag.Int("max-text",5000,"text character limit for extracted snapshot")
    smoke:=flag.Bool("local-smoke",false,"test only a local server, with no public internet access")
    extra:=flag.String("extra-hosts","","additional trusted hosts needed for redirects (comma-separated)")
    flag.Parse()
    if *repeat<1 || *repeat>10 {fmt.Fprintln(os.Stderr,"-repeat must be 1..10");os.Exit(2)}
    if *maxText<40 || *maxText>100000 {fmt.Fprintln(os.Stderr,"-max-text must be 40..100000");os.Exit(2)}
    var urls []string
    var cleanup func()
    mode:="public_web"
    if *smoke {
        srv,addr:=defaultSmoke();cleanup=srv.Close;urls=[]string{addr};mode="local_smoke"
    } else {
        var err error
        urls,err=readSites(*sites)
        if err!=nil {fmt.Fprintln(os.Stderr,err);os.Exit(2)}
    }
    if cleanup!=nil {defer cleanup()}
    extraHosts:=[]string{}
    for _,h:=range strings.Split(*extra,",") {if x:=strings.TrimSpace(h);x!="" {extraHosts=append(extraHosts,x)}}
    report:=execute(urls,*repeat,*worker,*maxText,mode,extraHosts)
    bytes,err:=json.MarshalIndent(report,"","  ")
    if err!=nil {fmt.Fprintln(os.Stderr,err);os.Exit(1)}
    if err=os.MkdirAll(filepath.Dir(*output),0755);err!=nil {fmt.Fprintln(os.Stderr,err);os.Exit(1)}
    if err=os.WriteFile(*output,append(bytes,'\n'),0644);err!=nil {fmt.Fprintln(os.Stderr,err);os.Exit(1)}
    fmt.Printf("mode=%s runs=%d useful=%d report=%s\n",mode,len(report.Results),report.UsefulPages,*output)
}
