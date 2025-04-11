"use strict";

function update() {
    const $files = document.getElementById("files");
    const $filter = document.getElementById("filter");

    const query = $filter.value
        .trim()
        .toLowerCase()
        .split(" ")
        .map((s) => s.trim())
        .filter(Boolean);

    for (let i = 0; i < $files.children.length; i++) {
        const queryStr = $files.children[i].dataset.queryStr;
        let match = true;
        for (let j = 0; j < query.length; j++) {
            if (!queryStr.includes(query[j])) {
                match = false;
                break;
            }
        }
        if (match) {
            $files.children[i].removeAttribute("hidden");
        } else {
            $files.children[i].setAttribute("hidden", "true");
        }
    }
}

document.getElementById("filter").addEventListener("input", function (event) {
    update();
});

update();
