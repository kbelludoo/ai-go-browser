"""
Example: Using AI Go Browser in an LLM pipeline.
"""

from aigo_browser import Browser

def main():
    browser = Browser()

    target_url = "https://news.ycombinator.com"
    print(f"🌐 Navigating to {target_url} using native Go engine...")
    page = browser.navigate(target_url)

    print("\n✅ Navigation Successful!")
    print(f"• Title:          {page.title}")
    print(f"• Status:         {page.status}")
    print(f"• Memory RSS:     {page.rss_mb:.2f} MiB (vs 750MB+ in Chromium)")
    print(f"• Duration:       {page.duration_ms:.1f} ms")
    print(f"• Raw Tokens:     {page.raw_tokens_est}")
    print(f"• Agent Tokens:   {page.agent_tokens_est} (-{page.token_savings_pct:.1f}% reduction)")
    print(f"• Indexed Links:  {len(page.links)} links mapped ([l1], [l2]...)")
    print(f"• Action Buttons: {len(page.buttons)} buttons with deduced endpoints")

if __name__ == "__main__":
    main()
