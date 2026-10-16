package browser

import "strconv"

// CSS selectors are evaluated independently in the document and each open
// shadow root. They do not turn descendant combinators into cross-root paths.
// Keep traversal bounded and stream elements rather than building an all-DOM
// array for every action or selector lookup.
const liveDOMHelpers = `const liveDOM = (() => {
	const parent = element => element.assignedSlot || element.parentElement || (element.getRootNode() instanceof ShadowRoot ? element.getRootNode().host : null);
	const prefix = (text, limit) => Array.from((text || '').slice(0, limit*2)).slice(0, limit).join('');
	const label = element => {
		let text = (element.getAttribute('aria-label') || '').trim();
		if (!text) {
			const root = element.getRootNode();
			for (const match of (element.getAttribute('aria-labelledby') || '').matchAll(/\S+/g)) {
				const source = root.getElementById && root.getElementById(match[0]);
				if (source) text += (text ? ' ' : '') + prefix(source.textContent,512);
				if (text.length >= 1024) break;
			}
			text = text.trim();
		}
		if (!text && element.labels) for (const source of element.labels) {
			text += (text ? ' ' : '') + prefix(source.textContent,512);
			if (text.length >= 1024) break;
		}
		return prefix(text || element.getAttribute('placeholder') || '',512);
	};
	const aria = (element, name) => {
		for (let current = element; current; current = parent(current)) {
			if ((current.getAttribute(name) || '').toLowerCase() === 'true') return true;
		}
		return false;
	};
	const disabled = element => element.matches(':disabled') || aria(element, 'aria-disabled');
	const readonly = element => !!element.readOnly || aria(element, 'aria-readonly');
	const role = (element, name) => (element.getAttribute('role') || '').toLowerCase().split(/\s+/).includes(name);
	const editable = element => element.isContentEditable && (!element.parentElement || !element.parentElement.isContentEditable);
	const genericButton = element => !['INPUT','TEXTAREA','SELECT','BUTTON'].includes(element.tagName) && (role(element, 'button') || (element.tagName === 'A' && !element.hasAttribute('href')));
	function* roots(root = document, budget = {count:0}, depth = 0) {
		if (depth > 512) throw new Error('live DOM exceeds 512 shadow levels');
		yield root;
		const walker = document.createTreeWalker(root, NodeFilter.SHOW_ELEMENT);
		let element;
		while (element = walker.nextNode()) {
			if (++budget.count > 50000) throw new Error('live DOM exceeds 50000 elements');
			if (element.shadowRoot) yield* roots(element.shadowRoot, budget, depth+1);
		}
	}
	const query = (selector, limit = 32) => {
		const results = [];
		for (const root of roots()) {
			for (const element of root.querySelectorAll(selector)) {
				results.push(element);
				if (results.length >= limit) return results;
			}
		}
		return results;
	};
	const visible = element => {
		if (!element || !element.isConnected) return false;
		const style = getComputedStyle(element), rect = element.getBoundingClientRect();
		return style.display !== 'none' && !['hidden','collapse'].includes(style.visibility) && rect.width > 0 && rect.height > 0;
	};
	const hit = (x, y) => {
		let element = document.elementFromPoint(x, y);
		for (let depth = 0; element && element.shadowRoot && depth < 512; depth++) {
			const nested = element.shadowRoot.elementFromPoint(x, y);
			if (!nested || nested === element) break;
			element = nested;
		}
		return element;
	};
	const contains = (container, element) => {
		for (let current = element; current; current = parent(current)) if (current === container) return true;
		return false;
	};
	return {parent, prefix, label, aria, disabled, readonly, role, editable, genericButton, roots, query, visible, hit, contains};
})();`

// This observation serializes the rendered, composed tree. Assigned slot nodes
// occur once; unused light DOM does not masquerade as visible page content.
// Open shadow roots are readable without changing the site's attachShadow API.
const liveSnapshotScript = `(() => {
` + liveDOMHelpers + `
	if (!window.__aiGoBrowserLive) window.__aiGoBrowserLive = {ids: new WeakMap(), next: 0};
	const state = window.__aiGoBrowserLive;
	const mark = element => {
		let id = state.ids.get(element);
		if (!id) { id = String(++state.next); state.ids.set(element, id); }
		if (element.getAttribute('data-aigobrowser') !== id) element.setAttribute('data-aigobrowser', id);
		return id;
	};
	const original = document.documentElement;
	if (!original) throw new Error('live page has no document element');
	let count = 0, editableValueBytes = 0;
	const inertTags = new Set(['HEAD','SCRIPT','STYLE','NOSCRIPT','TEMPLATE','SVG']);
	const copyNode = (node, inert = false, depth = 0) => {
		if (depth > 512) throw new Error('live DOM exceeds 512 levels');
		if (node.nodeType !== Node.ELEMENT_NODE) return node.cloneNode(true);
		if (++count > 50000) throw new Error('live DOM exceeds 50000 elements');
		const element = node;
		const control = ['INPUT','TEXTAREA','SELECT','BUTTON'].includes(element.tagName);
		if (control || element.matches('a[href],form') || liveDOM.editable(element) || liveDOM.role(element, 'textbox') || liveDOM.genericButton(element)) mark(element);
		// Nested forms are valid across shadow boundaries, but an HTML parser
		// would discard the inner <form> after flattening. Restore this tag in
		// the Go observation only, after parsing the serialized tree.
		const shadowForm = element instanceof HTMLFormElement && element.getRootNode() instanceof ShadowRoot;
		const copy = shadowForm ? document.createElement('aigobrowser-form') : element.cloneNode(false);
		if (shadowForm) for (const attr of element.attributes) copy.setAttribute(attr.name, attr.value);
		inert = inert || inertTags.has(element.tagName);
		if (!inert) {
			const style = getComputedStyle(element);
			if (style.display === 'none' || ['hidden','collapse'].includes(style.visibility)) copy.setAttribute('hidden', '');
		}
		if (control) {
			copy.setAttribute('data-aigobrowser-form', element.form ? mark(element.form) : '');
			if (liveDOM.disabled(element)) copy.setAttribute('aria-disabled', 'true');
			if (liveDOM.readonly(element)) copy.setAttribute('aria-readonly', 'true');
		}
		if (control || liveDOM.editable(element) || liveDOM.role(element, 'textbox') || liveDOM.genericButton(element)) copy.setAttribute('data-aigobrowser-label', liveDOM.label(element));
		// Preserve rendered line breaks without flattening the editor's child
		// controls or guessing layout from its serialized paragraph/BR tags.
		if (!control && (liveDOM.editable(element) || liveDOM.role(element, 'textbox'))) {
			const value = element.innerText || '', remaining = 5242880 - editableValueBytes;
			if (value.length > remaining) throw new Error('live editable values exceed 5242880 bytes');
			const bytes = new Blob([value]).size;
			if (bytes > remaining) throw new Error('live editable values exceed 5242880 bytes');
			editableValueBytes += bytes;
			copy.setAttribute('data-aigobrowser-editable-value', value);
		}
		if (element instanceof HTMLInputElement) {
			copy.setAttribute('value', element.value);
			if (element.checked) copy.setAttribute('checked', ''); else copy.removeAttribute('checked');
		}
		if (element instanceof HTMLTextAreaElement) {
			copy.textContent = element.value;
		} else {
			const children = element.shadowRoot ? element.shadowRoot.childNodes :
				element instanceof HTMLSlotElement && element.assignedNodes().length ? element.assignedNodes({flatten:true}) : element.childNodes;
			for (const child of children) copy.appendChild(copyNode(child, inert, depth+1));
		}
		if (element instanceof HTMLOptionElement) {
			if (element.selected) copy.setAttribute('selected', ''); else copy.removeAttribute('selected');
		}
		return copy;
	};
	const clone = copyNode(original);
	const html = '<!doctype html>' + clone.outerHTML;
	if (html.length > 5242880 || new Blob([html]).size > 5242880) throw new Error('live DOM exceeds 5242880 bytes');
	return {html, url: location.href, base: document.baseURI};
})()`

// liveReadScript returns only the observation an agent reads while a page is
// settling. It deliberately skips cloning, marking and serializing the DOM:
// that work is the cost liveRead exists to avoid, and none of it produces the
// visible text this view reports.
//
// The text goes through the same normalization a full observation applies
// (every whitespace run collapses to one space, nothing is left dangling at
// either end), so a read and a full observation report the same shape and an
// agent can poll with one and act on the other. innerText is used because it
// reads the rendered tree, which is what the read view promises.
//
// The cap is in Unicode code points to match the Go parser's limit, and
// surrogate pairs are never split.
func liveReadScript(limit int) string {
	return `(() => {
		const body = document.body;
		const raw = body ? (body.innerText || '') : '';
		const cap = ` + strconv.Itoa(limit) + `;
		const points = [];
		let space = false;
		let overflow = false;
		for (let i = 0; i < raw.length; ) {
			if (points.length >= cap) { overflow = true; break; }
			const cp = raw.codePointAt(i);
			i += cp > 0xFFFF ? 2 : 1;
			const ch = String.fromCodePoint(cp);
			if (/\s/.test(ch)) { space = points.length > 0; continue; }
			if (space) {
				points.push(' ');
				space = false;
				if (points.length >= cap) { overflow = true; break; }
			}
			points.push(ch);
		}
		const navigation = performance.getEntriesByType('navigation')[0];
		return {
			url: location.href,
			base: document.baseURI,
			title: document.title || '',
			text: points.join(''),
			truncated: overflow,
			status: (navigation && navigation.responseStatus) || 0
		};
	})()`
}
