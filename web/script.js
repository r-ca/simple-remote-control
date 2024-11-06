// 初期化処理
document.addEventListener("DOMContentLoaded", function() {
    const addButton = document.getElementById("addButton");
    const prevSlideButton = document.getElementById("prevSlide");
    const nextSlideButton = document.getElementById("nextSlide");

    addButton.addEventListener("click", addAddress);
    prevSlideButton.addEventListener("click", () => changeSlide('prev'));
    nextSlideButton.addEventListener("click", () => changeSlide('next'));
});

let addressList = [];

// アドレスを追加する関数
function addAddress() {
    const addressInput = document.getElementById("addressInput");
    const rawAddress = addressInput.value.trim();

    const address = formatAddress(rawAddress);
    if (!address) {
        alert("有効なアドレスを入力してください。");
        return;
    }

    const tableBody = document.getElementById("addressTableBody");
    const row = createAddressRow(address);
    tableBody.appendChild(row);

    // アドレスリストに追加
    addressList.push(address);

    addressInput.value = "";
}

// アドレスをフォーマット・バリデーションする関数
function formatAddress(address) {
    if (!address) return null;

    // プロトコルがない場合は http:// を追加
    if (!/^https?:\/\//i.test(address)) {
        address = `http://${address}`;
    }

    // 末尾のスラッシュを削除
    address = address.replace(/\/+$/, "");

    // 簡易URLバリデーション
    const urlPattern = /^(https?:\/\/)?([a-zA-Z0-9-]+(\.[a-zA-Z0-9-]+)*)(:\d+)?(\/.*)?$/;
    if (!urlPattern.test(address)) return null;

    return address;
}

// アドレステーブルの行を作成する関数
function createAddressRow(address) {
    const row = document.createElement("tr");

    const addressCell = document.createElement("td");
    addressCell.textContent = address;
    row.appendChild(addressCell);

    const statusCell = document.createElement("td");
    const statusIcon = document.createElement("span");
    statusIcon.classList.add("status-icon");
    statusCell.appendChild(statusIcon);
    row.appendChild(statusCell);

    const actionCell = document.createElement("td");
    const deleteButton = document.createElement("span");
    deleteButton.innerHTML = '<i class="fas fa-trash-alt"></i>';
    deleteButton.classList.add("delete-button");
    deleteButton.onclick = () => {
        // アドレスリストから削除
        addressList = addressList.filter(item => item !== address);
        // テーブルから削除
        row.remove();
    }
    actionCell.appendChild(deleteButton);
    row.appendChild(actionCell);

    return row;
}

// プレゼンテーションのスライドを変更する関数
function changeSlide(direction) {
    sendKeyPress(direction === 'prev' ? 'left' : 'right');
}

// TODO: 内部でグローバル変数を参照するのをやめたい
function checkStatus() {
    addressList.forEach((address, index) => {
        fetch(address)
            .then(response => {
                if (response.ok) {
                    console.log(`[${index}] ${address} is OK`);
                    updateStatus(address, true);
                } else {
                    console.error(`[${index}] ${address} is NOT OK`);
                    updateStatus(address, false);
                }
            })
            .catch(error => {
                console.error(`[${index}] ${address} is NOT OK`);
                updateStatus(address, false);
            });
    });
}

function updateStatus(address, status) {
    const tableBody = document.getElementById("addressTableBody");
    const rows = tableBody.getElementsByTagName("tr");
    for (let i = 0; i < rows.length; i++) {
        const row = rows[i];
        const addressCell = row.getElementsByTagName("td")[0];
        if (addressCell.textContent === address) {
            const statusCell = row.getElementsByTagName("td")[1];
            const statusIcon = statusCell.getElementsByClassName("status-icon")[0];
            statusIcon.classList.remove("success", "error");
            statusIcon.classList.add(status === true ? "success" : "error");
            break;
        }
    }
}

// 定期的にステータスをチェックする
setInterval(checkStatus, 5000);

// Supported keys: "left", "right"
function sendKeyPress(key) {
    addressList.forEach((address, index) => {
        fetch(`${address}/press_key`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ key })
        })
            .then(response => response.ok ? response.text() : Promise.reject(response.statusText))
            .then(console.log)
            .catch(console.error);
    });
}

