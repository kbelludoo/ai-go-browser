package main

import "core:strings"
import "core:net"

// Build an explicit GET-only form submission URL. This does not send an
// HTTP request. POST and other methods are intentionally unsupported.
form_get_url :: proc(form: Parsed_Form, overrides: []string) -> (string, bool) {
	if form.method != "get" || form.action == "" || !fetch_validate_url(form.action) {
		return "", false
	}
	for item in overrides {
		equals := strings.index(item, "=")
		if equals <= 0 {
			return "", false
		}
		field_name := item[:equals]
		found := false
		for field in form.fields {
			if field.name == field_name && !field.disabled &&
				field.type != "submit" && field.type != "button" && field.type != "file" {
				found = true
				break
			}
		}
		if !found {return "", false}
	}

	out := strings.builder_make()
	defer strings.builder_destroy(&out)
	action := form.action
	if idx := strings.index(action, "#"); idx >= 0 {
		action = action[:idx]
	}
	strings.write_string(&out, action)
	separator: byte = '?'
	if strings.contains(action, "?") {
		separator = '&'
	}
	for field in form.fields {
		if field.disabled || field.name == "" || field.type == "submit" ||
			field.type == "button" || field.type == "file" || field.type == "reset" {
			continue
		}
		if (field.type == "checkbox" || field.type == "radio") && !field.checked {
			continue
		}
		value := field.value
		for item in overrides {
			eq := strings.index(item, "=")
			if item[:eq] == field.name {
				value = item[eq+1:]
			}
		}
		name_encoded := net.percent_encode(field.name)
		value_encoded := net.percent_encode(value)
		strings.write_byte(&out, separator)
		separator = '&'
		strings.write_string(&out, name_encoded)
		strings.write_byte(&out, '=')
		strings.write_string(&out, value_encoded)
	}
	return strings.clone(strings.to_string(out)), true
}
