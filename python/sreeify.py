# import re
# import logging
# import time
#
# from bs4 import BeautifulSoup
from itertools import zip_longest
import logging
from lxml import etree, html

import gen.sreeify_pb2 as sreeify_pb2



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

EXC_TAGS = ["script", "style", "head", "title", "meta", "link", "noscript", "script"]
REQ_TAGS = ["body"]  # Assuming you want to process all tags within <body>
XPATH_EXPR = f"//{'|//'.join(REQ_TAGS)}//*[not(self::{' or self::'.join(EXC_TAGS)})]/text()"
logging.info(f"XPath expression: {XPATH_EXPR}")


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


def sreeify_text_lxml(payload: str, link_replacements: list) -> str:
    def replace_links(s: str) -> str:
        for link in link_replacements:
            s = s.replace(link.original, link.replacement)
        return s

    tree = html.fromstring(payload)
    tree.rewrite_links(replace_links)

    for text_node in tree.xpath(XPATH_EXPR):
        parent = text_node.getparent()

        if text_node.is_text:
            parent.text = split_n_join(sreeify_word, parent.text)
            parent.tail = split_n_join(sreeify_word, parent.tail)

    return etree.tostring(tree, pretty_print=True, method="html", encoding='unicode')


def sreeify_text(payload: str, link_replacements: list) -> str:
    return sreeify_text_lxml(payload, link_replacements)


def sreeify_word(word: str) -> str:
    return ("sree" * -(-len(word)//4))[:len(word)]
