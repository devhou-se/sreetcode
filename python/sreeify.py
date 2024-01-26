import re

from bs4 import BeautifulSoup


WORD_REPLACENTS = {
    "Wiki": "Sreeki",
    "free encyclopedia": "Sree encyclopedia",
    "Encyclopedia": "Encyclosreedia",
    "Free ": "Sree ",
    "Free_": "Sree_",
    "Free<": "Sree<",
    "Media": "Sreedia",
}

URL_MAPPINGS = {
    "https://en.wikipedia.org/": "/",
    "https://en.wiktionary.org/": "/dict/",
    "https://en.sreekinews.org/": "/news/",
    "https://en.sreekiquote.org/": "/quote/",
    "https://en.sreekiversity.org/": "/sreekiversity/",
}

def sreeify(payload: str) -> str:
    doc = BeautifulSoup(payload, "html.parser")

    # imgs = doc.find_all("img")
    # for img in imgs:
    #     img["src"] = "https://i.imgur.com/6H6i6qQ.jpg"
    #     img["srcset"] = ""

    body = doc.find("body")
    for original, replaced in WORD_REPLACENTS.items():
        o1, r1 = original.lower(), replaced.lower()
        o2, r2 = original.upper(), replaced.upper()
        # o3, r3 = original.capitalize(), replaced.capitalize()
        o4, r4 = original, replaced

        for o, r in [(o1, r1), (o2, r2), (o4, r4)]:

            candidates = body.find_all(string=re.compile(o))
            for candidate in candidates:
                candidate.replace_with(candidate.replace(o, r))

    return str(doc)


# // Unsreefy reverses the replacements made by the Sreefy function, restoring the original words.
# func Unsreefy(input string) string {
# 	// Replace each occurrence of the 'value' with its corresponding 'key'.
# 	for original, replaced := range wordReplacements {
# 		// Replace with respect to case variations (normal, lower, upper).
# 		input = strings.ReplaceAll(input, replaced, original)
# 		input = strings.ReplaceAll(input, strings.ToLower(replaced), strings.ToLower(original))
# 		input = strings.ReplaceAll(input, strings.ToUpper(replaced), strings.ToUpper(original))
# 	}
#
# 	return input
# }
#
# // Sreefy performs a set of replacements within the input string according to the wordReplacements map.
# func Sreefy(input string) string {
# 	// Regular expression to identify URLs to be temporarily removed from the replacement process.
# 	urlPattern := regexp.MustCompile(`(//)((\w+)\.wikimedia.org)([\w\d+/_\-.%]*)["\s]`)
# 	urlMatches := urlPattern.FindAllString(input, -1)
#
# 	// Temporarily mask matched URLs using a placeholder.
# 	for i, match := range urlMatches {
# 		placeholder := fmt.Sprintf("{{%d}}", i)
# 		input = strings.ReplaceAll(input, match, placeholder)
# 	}
#
# 	// Perform word replacements for different case variations (normal, lower, upper).
# 	for original, replaced := range wordReplacements {
# 		input = strings.ReplaceAll(input, original, replaced)
# 		input = strings.ReplaceAll(input, strings.ToLower(original), strings.ToLower(replaced))
# 		input = strings.ReplaceAll(input, strings.ToUpper(original), strings.ToUpper(replaced))
# 	}
#
# 	// Correct specific misreplacements.
# 	input = strings.ReplaceAll(input, "matchSreedia", "matchMedia")
# 	input = strings.ReplaceAll(input, "@sreedia", "@media")
#
# 	// Restore the original URLs by replacing the placeholders.
# 	for i, match := range urlMatches {
# 		placeholder := fmt.Sprintf("{{%d}}", i)
# 		input = strings.ReplaceAll(input, placeholder, match)
# 	}
#
# 	return input
# }
#
# func UpdateURLs(body string) string {
# 	for original, replaced := range URLMappings {
# 		body = strings.ReplaceAll(body, original, replaced)
# 	}
#
# 	return body
# }