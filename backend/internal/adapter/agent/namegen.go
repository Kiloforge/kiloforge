package agent

import "math/rand/v2"

var adverbs = []string{
	"curiously", "remarkably", "suspiciously", "absurdly", "delightfully",
	"genuinely", "mildly", "secretly", "oddly", "fiercely",
	"quietly", "boldly", "cheerfully", "cautiously", "wildly",
	"strangely", "perfectly", "barely", "incredibly", "slightly",
	"terribly", "wonderfully", "mysteriously", "ridiculously", "pleasantly",
	"dangerously", "enthusiastically", "unusually", "hilariously", "profoundly",
}

var adjectives = []string{
	"dangerous", "cheerful", "quiet", "brave", "clever",
	"sleepy", "hungry", "swift", "gentle", "fierce",
	"witty", "bold", "calm", "eager", "grumpy",
	"jolly", "keen", "lazy", "noble", "proud",
	"shy", "tall", "wise", "zany", "crafty",
	"daring", "elegant", "fearless", "graceful", "hasty",
	"inventive", "jubilant", "kind", "lively", "mighty",
	"nimble", "optimistic", "patient", "quirky", "resilient",
	"sneaky", "tenacious", "upbeat", "valiant", "whimsical",
	"adventurous", "brilliant", "cosmic", "determined", "electric",
}

var names = []string{
	"ada", "blake", "cleo", "dave", "eve",
	"finn", "grace", "hank", "iris", "jules",
	"kai", "luna", "max", "nova", "oscar",
	"piper", "quinn", "ruby", "sage", "tara",
	"uma", "vex", "wren", "xena", "yuri",
	"zara", "arrow", "blaze", "cedar", "drift",
	"echo", "fern", "glow", "haze", "ink",
	"jade", "kite", "lark", "moss", "nyx",
	"onyx", "pixel", "rain", "spark", "tide",
	"volt", "wave", "zen", "atlas", "byte",
}

// GenerateName returns a random human-friendly name in "adverb adjective name" format.
func GenerateName() string {
	a := adverbs[rand.IntN(len(adverbs))]
	b := adjectives[rand.IntN(len(adjectives))]
	c := names[rand.IntN(len(names))]
	return a + " " + b + " " + c
}
