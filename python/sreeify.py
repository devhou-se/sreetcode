from itertools import zip_longest
import logging

from lxml import etree, html


WORD_REPLACEMENTS = {
    # "Wiki": "Sreeki",
    # "free encyclopedia": "Sree encyclopedia",
    # "Encyclopedia": "Encyclosreedia",
    # "Free ": "Sree ",
    # "Free_": "Sree_",
    # "Free<": "Sree<",
    # "Media": "Sreedia",
    "wiki": "sreeki",
    "wikipedia": "sreekipedia",
    "encyclopedia": "encyclosreedia",
    "free": "sree",
    "media": "sreedia",
}

URL_MAPPINGS = {
    "https://en.wikipedia.org/": "/",
    "https://en.wiktionary.org/": "/dict/",
    "https://en.sreekinews.org/": "/news/",
    "https://en.sreekiquote.org/": "/quote/",
    "https://en.sreekiversity.org/": "/sreekiversity/",
}

EXC_TAGS = ["script", "style", "head", "title", "meta", "link", "noscript", "script"]
REQ_TAGS = ["body"]
XPATH_EXPR = f"//{'|//'.join(REQ_TAGS)}//*[not(self::{' or self::'.join(EXC_TAGS)})]/text()"


def split_n_join(m: callable, s: str) -> str:
    if not s:
        return s

    seps = [[], []]

    curr = ""
    start_alpha = s[0].isalpha()
    alpha = start_alpha

    for c in s:
        if c.isalpha() != alpha:
            seps[alpha].append(curr)
            curr = ""
            alpha = not alpha
        curr += c
    seps[alpha].append(curr)
    seps[1] = [m(w) for w in seps[1]]

    if start_alpha:
        b, a = seps
    else:
        a, b = seps

    return "".join([m + n for m, n in zip_longest(a, b, fillvalue="")])


def sreeify_text_lxml(payload: str) -> str:
    def replace_links(s: str) -> str:
        if s.startswith("/wiki/"):
            s = "/sreeki/" + s[6:]

        return s

    tree = html.fromstring(payload)
    tree.rewrite_links(replace_links)

    # TODO: FIX: This block isn't capturing all the text nodes
    for text_node in tree.xpath(XPATH_EXPR):
        parent = text_node.getparent()

        if text_node.is_text:
            parent.text = split_n_join(sreeify_word, parent.text)
            parent.tail = split_n_join(sreeify_word, parent.tail)

    return etree.tostring(tree, pretty_print=True, method="html", encoding='unicode')


def sreeify_text(payload: str) -> str:
    return sreeify_text_lxml(payload)


def sreeify_word(word: str) -> str:
    for w, r in [("wiki", "sreeki"), ("Wiki", "Sreeki"), ("WIKI", "SREEKI")]:
        if word.startswith(w):
            return r + word[len(w):]

    for k, v in WORD_REPLACEMENTS.items():
        if word == k:
            return v
        if word == k.capitalize():
            return v.capitalize()
        if word == k.upper():
            return v.upper()
        if word == k.lower():
            return v.lower()
    return word
