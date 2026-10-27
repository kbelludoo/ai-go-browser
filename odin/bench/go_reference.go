// Go reference for a reproducible native Go vs Odin *static* page benchmark.
// It imports the actual browser package, not an imitation of its parser.
// Scope: parse HTML and compact CSO. This DOES NOT benchmark JS, CDP, HTTP or TLS.
package main

import (
 "encoding/json"
 "fmt"
 "os"
 "strconv"
 "strings"

 "ai-go-browser/browser"
)

type Heading struct { Level int `json:"level"`; Text string `json:"text"` }
type Link struct { Label string `json:"label"`; URL string `json:"url"` }
type Projection struct {
 Title string `json:"title"`
 Description string `json:"description"`
 Text string `json:"text"`
 Headings []Heading `json:"headings"`
 Links []Link `json:"links"`
 CSO string `json:"cso"`
}
func project(p browser.Page) Projection {
 result := Projection{Title: p.Title, Description: p.Description, Text: p.Text, CSO: p.ToCSO(15000),
   Headings: make([]Heading,0,len(p.Headings)), Links: make([]Link,0,len(p.Links))}
 for _, h := range p.Headings { result.Headings = append(result.Headings, Heading{h.Level,h.Text}) }
 for _, l := range p.Links { result.Links = append(result.Links, Link{l.Label,l.URL}) }
 return result
}
func main() {
 if len(os.Args) < 4 {
  fmt.Fprintln(os.Stderr, "usage: go-reference parse-file|bench-file path url [iterations]")
  os.Exit(2)
 }
 data,err:=os.ReadFile(os.Args[2]); if err!=nil {panic(err)}
 action,url:=os.Args[1],os.Args[3]
 switch action {
 case "parse-file":
  p:=browser.Parse(data,url,200,len(data),0)
  if err:=json.NewEncoder(os.Stdout).Encode(project(p));err!=nil {panic(err)}
 case "bench-file":
  if len(os.Args)!=5 {panic("iterations expected")}
  n,err:=strconv.Atoi(os.Args[4]);if err!=nil||n<1||n>200000 {panic("bad iterations")}
  sum:=0
  for i:=0;i<n;i++ {
   p:=browser.Parse(data,url,200,len(data),0)
   s:=p.ToCSO(15000)
   sum+=len(p.Text)+len(p.Title)+len(s)
  }
  fmt.Printf("checksum=%d iterations=%d runtime=Go\n",sum,n)
 case "tokens":
  fmt.Println(browser.EstimateTokens(string(data)))
 case "markdown":
  fmt.Println(strings.TrimSpace(browser.MarkdownToCSO(string(data),10000)))
 default: panic("unknown action")
 }
}
