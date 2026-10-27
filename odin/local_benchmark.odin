package main

import "core:fmt"
import "core:os"
import "core:strconv"
import "core:encoding/json"
import vmem "core:mem/virtual"

// Benchmark ONLY the common static parse + CSO operations from browser/cso.go
// and browser/parser.go. Read source before timing, and keep each iteration's
// arena isolated. No network stack, persistent browser or JS in this benchmark.
Static_Projection :: struct {
	title: string,
	description: string,
	text: string,
	headings: []Heading,
	links: []Page_Link,
	cso: string,
}

benchmark_file :: proc(action, filepath, url: string, repetitions: int = 1) {
	content, err := os.read_entire_file(filepath, context.allocator)
	if err != nil {
		fmt.printf("Cannot read benchmark file: %v\n", err)
		return
	}
	defer delete(content)

	sum := 0
	for i in 0..<repetitions {
		arena: vmem.Arena
		if vmem.arena_init_growing(&arena) != nil {return}
		{
			context.allocator = vmem.arena_allocator(&arena)
			page := parse_html(string(content), url)
			cso := page_to_cso(page.semantic)
			if action == "parse-file" {
				value := Static_Projection{
					title=page.semantic.title,
					description=page.semantic.description,
					text=page.semantic.text,
					headings=page.semantic.headings,
					links=page.semantic.links,
					cso=cso,
				}
				data, json_err := json.marshal(value)
				if json_err != nil {fmt.println("JSON marshal failed")}
				if json_err == nil {fmt.println(string(data))}
			}
			sum += len(page.semantic.text)+len(page.semantic.title)+len(cso)
		}
		vmem.arena_destroy(&arena)
	}
	if action == "bench-file" {
		fmt.printf("checksum=%d iterations=%d runtime=Odin\n",sum,repetitions)
	}
}

benchmark_cli :: proc(args: []string) -> bool {
	if len(args)<2 {return false}
	action := args[1]
	if action != "parse-file" && action != "bench-file" {return false}
	if (action == "parse-file" && len(args)!=4) ||
		(action == "bench-file" && len(args)!=5) {
		fmt.println("Use: ai-odin-browser parse-file|bench-file <html-file> <url> [iterations]")
		return true
	}
	repetitions := 1
	if action=="bench-file" {
		n,ok := strconv.parse_int(args[4])
		if !ok || n < 1 || n > 200000 {
			fmt.println("Invalid iteration count [1..200000]")
			return true
		}
		repetitions=n
	}
	benchmark_file(action,args[2],args[3],repetitions)
	return true
}
