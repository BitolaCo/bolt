/*
If you want to use this JavaScript directly, replace the "%s" at the end
of the file with the appropriate values.
*/
(function(proxy, cl) {
    var elements = document.getElementsByClassName(cl);
    for (var i = 0; i < elements.length; i++) {
        var el = elements[i],
            src = el.getAttribute("data-src"),
            bg = el.getAttribute("data-bg");
        if (src) {
            el.src = ["/", proxy, el.offsetWidth, src].join("/");
        }
        if (bg) {
            el.style.backgroundImage = "url(" + ["/", proxy, el.offsetWidth, bg].join("/") + ")";
        }
    }
}("%s", "%s"));