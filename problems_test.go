package main

import "testing"

func TestHTMLToTextRemovesTags(t *testing.T) {
	got := htmlToText("<p>You are given an <strong>array</strong>.</p>")

	want := "You are given an array."
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestHTMLToTextFormatsProblemContent(t *testing.T) {
	got := htmlToText(`<p><strong>Example 1:</strong></p><pre><strong>Input:</strong> nums = [2,7], target = 9</pre><ul><li><code>2 &lt;= nums.length &lt;= 10<sup>4</sup></code></li></ul>`)

	want := "Example 1:\n\nInput: nums = [2,7], target = 9\n\n- 2 <= nums.length <= 10^4"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
