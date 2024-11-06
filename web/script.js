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
    deleteButton.onclick = () => row.remove();
    actionCell.appendChild(deleteButton);
    row.appendChild(actionCell);

    return row;
}

// プレゼンテーションのスライドを変更する関数
function changeSlide(direction) {
    if (direction === 'prev') {
        console.log("前のスライドに移動します");
        // 前のスライドに移動する処理
    } else if (direction === 'next') {
        console.log("次のスライドに移動します");
        // 次のスライドに移動する処理
    }
}


