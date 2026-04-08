// 定义地址
const myIP = "172.20.10.3"
// 定义端口
const port = "8080"

// 切换模块的功能
const buttons = document.querySelectorAll(".sidebar button");
const modules = document.querySelectorAll(".module");

// 展示不同模块
buttons.forEach(button => {
    button.addEventListener("click", () => {
        // 隐藏所有模块
        modules.forEach(module => module.classList.add("hidden"));
        modules.forEach(module => module.classList.remove("active"));
        // 显示目标模块
        const target = button.getAttribute("data-target");
        document.getElementById(target).classList.remove("hidden");
    });
});

// 创建节点
document.querySelector(".node-form").addEventListener("submit", async function (event) {
    event.preventDefault();

    const nodeName = document.querySelector(".node-input").value.trim();
    const responseElement = document.querySelector(".response");

    if (!nodeName) {
        responseElement.innerText = "请输入节点名称！";
        responseElement.className = "response error";
        return;
    }

    try {
        const response = await fetch("http://" + myIP + ":" + port + "/create-node", {
            method: "POST", headers: {
                "Content-Type": "application/json"
            }, body: JSON.stringify({name: nodeName})
        });

        const result = await response.json();
        if (response.ok) {
            responseElement.innerText = `节点创建成功！地址：${result.address + '/p2p/' + result.nodeID}`;
            responseElement.className = "response success";
            addNodeToList(nodeName, result.address + "/p2p/" + result.nodeID)
        } else {
            responseElement.innerText = `错误：${result.error}`;
            responseElement.className = "response error";
        }
    } catch (error) {
        responseElement.innerText = `网络错误：${error.message}`;
        responseElement.className = "response error";
    }

});

// 连接节点
document.querySelector(".node-connect-form").addEventListener("submit", async function (event) {
    event.preventDefault();

    const nodeFrom = document.querySelector("#my-nodes").value
    const nodeTo = document.querySelector(".node-id").value.trim()
    const responseElement = document.querySelector(".warn");

    if (!nodeFrom) {
        responseElement.innerText = "请选择一个节点";
        responseElement.className = " warn error";
        return;
    }
    try {
        const response = await fetch("http://" + myIP + ":" + port + "/connect-node", {
            method: "POST", headers: {
                "Content-Type": "application/json"
            }, body: JSON.stringify({from: nodeFrom, to: nodeTo})
        });

        const result = await response.json();
        if (result.code === "1") {
            responseElement.innerText = "连接成功！";
            responseElement.className = "warn success";
        } else {
            responseElement.innerText = `${result.message}`
            responseElement.className = "warn error";
        }
    } catch (error) {
        responseElement.innerText = `网络错误：${error.message}`;
        responseElement.className = "warn error";
    }

});

// 发送消息
document.querySelector(".message").addEventListener("submit", async function (event) {
    event.preventDefault()

    const msg = document.querySelector("#chat-input").value
    const nodeFrom = document.querySelector("#my-nodes").value
    const nodeTo = document.querySelector(".node-id").value.trim()
    const responseElement = document.querySelector(".warn");

    if (msg === "") {
        alert("不允许发送空消息")
        return
    }
    if (!nodeFrom) {
        responseElement.innerText = "请选择一个节点";
        responseElement.className = " warn error";
        return;
    }
    try {
        const response = await fetch("http://" + myIP + ":" + port + "/send-msg", {
            method: "POST", headers: {
                "Content-Type": "application/json"
            }, body: JSON.stringify({from: nodeFrom, to: nodeTo, msg: msg})
        });
        const result = await response.json()
        if (result.code === "1") {
            document.querySelector("#chat-input").value = ""// 清空输入框内容
            alert("发送成功")
        } else {
            alert("发送失败！请检查是否有错误")
        }
    } catch (error) {

    }


})

// 显示其他节点发来的消息
document.querySelector(".refresh").addEventListener("submit", async function (event) {
    event.preventDefault()

    try {
        const response = await fetch("http://" + myIP + ":" + port + "/refresh-msg", {
            method: "POST", headers: {
                "Content-Type": "application/json"
            },
        })

        const result = await response.json()
        console.log(result)
        if (result.who !== "" || result.msg !== "") {
            document.querySelector("#read-text").value += (result.who + ":\t" + result.msg + '\n')
        }

    } catch (error) {

    }


})

// 向节点列表中添加节点信息 和 在下拉框中展示
function addNodeToList(name, address) {
    const nodeList = document.getElementById("node-list");

    // 创建 li 元素作为容器
    const listItem = document.createElement("li");

    // 创建 p 标签用于展示节点信息
    const nodeInfo = document.createElement("p");
    nodeInfo.textContent = name + "|----|" + address
    nodeInfo.style.margin = "5px 0";
    nodeInfo.style.padding = "10px";
    nodeInfo.style.border = "1px solid #ccc";
    nodeInfo.style.borderRadius = "5px";
    nodeInfo.style.backgroundColor = "#f8f9fa";
    nodeInfo.style.fontSize = "bold"

    // 将 p 添加到 li 中
    listItem.appendChild(nodeInfo);

    // 将 li 添加到 ul 中
    nodeList.appendChild(listItem);

    // 向实时聊天的下拉框中添加已创建的节点
    const myNodes = document.getElementById("my-nodes");
    const myNodesOption = document.createElement("option");
    myNodesOption.value = name
    myNodesOption.textContent = name.toString()
    // 保留意见。后续考虑改为 data-的自定属性。
    myNodesOption.id = name
    myNodes.appendChild(myNodesOption)

    // 向文件管理的下拉框中添加已创建的节点
    const myNodesInFileManage = document.getElementById("my-nodes-fileManage");
    const myNodesOptionInFileManage = document.createElement("option");
    myNodesOptionInFileManage.value = address
    myNodesOptionInFileManage.textContent = name.toString()
    myNodesOptionInFileManage.id = name
    myNodesInFileManage.appendChild(myNodesOptionInFileManage)

    // 向文件下载列表下拉框添加
    const myNodesInFileManageDownload = document.getElementById("my-nodes-fileManage-download")
    const myNodesOptionInFileManageDownload = document.createElement("option")
    myNodesOptionInFileManageDownload.value = address
    myNodesOptionInFileManageDownload.textContent = name
    myNodesOptionInFileManageDownload.id = name
    myNodesInFileManageDownload.appendChild(myNodesOptionInFileManageDownload)
}

// 文件管理模块文件选择按钮交互效果
const fileInput = document.getElementById("file-input");
const fileLabel = document.querySelector(".custom-file-label");
fileInput.addEventListener("change", () => {
    if (fileInput.files.length > 0) {
        fileLabel.textContent = `已选择：${fileInput.files[0].name}`;
    } else {
        fileLabel.textContent = "选择文件";
    }
});

//  上传文件
const form = document.querySelector(".file-upload-form")
form.addEventListener("submit", async function (event) {
    event.preventDefault()
    // 获取发送节点的信息
    const mySelect = document.querySelector("#my-nodes-fileManage")
    const mySelectIndex = mySelect.selectedIndex
    const nodeName = mySelect.options[mySelectIndex].id
    const nodeAddr = mySelect.options[mySelectIndex].value

    const formData = new FormData(form)
    formData.append("nodeName", nodeName)
    formData.append("nodeAddr", nodeAddr)
    console.log(formData.get("nodeName"))
    const response = await fetch("/upload-file", {
        method: "POST", body: formData,
    })
    if (response.ok) {
        alert("文件上传成功！")
    } else {
        alert("文件上传失败！")
    }
})

// 刷新文件列表，供用户下载
document.querySelector(".refresh-file-list-btn").addEventListener("click", async function (event) {
    event.preventDefault()

    const mySelect = document.querySelector("#my-nodes-fileManage-download")
    const mySelectIndex = mySelect.selectedIndex
    const nodeName = mySelect.options[mySelectIndex].id
    const nodeAddr = mySelect.options[mySelectIndex].value

    const formData = new FormData()
    formData.append("nodeName", nodeName)
    formData.append("nodeAddr", nodeAddr)
    try {
        const response = await fetch("http://" + myIP + ":" + port + "/refresh-file-list", {
            method: "POST",
            body: formData,
        })
        const result = await response.json()

        const fileDownloadList = document.querySelector(".file-download-list");
        fileDownloadList.innerHTML = ""; // 清空旧列表

        console.log(result)

        result.files.forEach(node => {
            // 显示节点名称
            const nodeTitle = document.createElement("h4");
            nodeTitle.textContent = `节点: ${node.node_name}`;
            fileDownloadList.appendChild(nodeTitle);

            // 创建文件列表
            const fileList = document.createElement("ul");
            node.files.forEach(file => {
                const listItem = document.createElement("li");

                // 文件名
                const fileName = document.createElement("span");
                fileName.textContent = file;

                // 下载按钮
                const downloadButton = document.createElement("button");
                downloadButton.textContent = "下载";
                downloadButton.className = "download-btn";
                downloadButton.style.marginLeft = "10px";
                downloadButton.value = node.ownerIP
                // downloadButton.name = node.nodeName

                // 为下载按钮添加事件监听器
                downloadButton.addEventListener("click", async () => {
                    console.log("下载 IP 为：", downloadButton.value)
                    try {
                        const response = await fetch(`http://${downloadButton.value}:${port}/download-file`, {
                            method: "POST",
                            headers: {
                                "Content-Type": "application/json"
                            },
                            body: JSON.stringify({
                                ownerIP: downloadButton.value,
                                nodeName: node.node_name,
                                fileName: file
                            }),
                            // mode: 'no-cors',  // 设置为 no-cors
                        });

                        console.log(response)

                        if (!response.ok) {
                            alert("文件下载失败！");
                            return;
                        }

                        // 获取文件内容并创建 Blob
                        const blob = await response.blob();

                        // 创建下载链接
                        const downloadLink = document.createElement("a");
                        downloadLink.href = URL.createObjectURL(blob);
                        downloadLink.download = file; // 设置文件名
                        document.body.appendChild(downloadLink);
                        downloadLink.click();
                        document.body.removeChild(downloadLink);
                    } catch (error) {
                        console.error("下载失败：", error);
                        alert("下载失败，请检查控制台！");
                    }
                });


                listItem.appendChild(fileName);
                listItem.appendChild(downloadButton);
                fileList.appendChild(listItem);
            });

            fileDownloadList.appendChild(fileList);
        });

    } catch (error) {
        console.log("刷新失败", error)
    }
})

// 展示现有节点
document.querySelector(".nodes-search-form").addEventListener('submit', async function (event) {
    event.preventDefault()

    const nodes_exist_list = document.querySelector(".nodes-exist-list")
    nodes_exist_list.innerHTML = ""


    try {
        const response = await fetch("http://" + myIP + ":" + port + "/exist-nodes", {
            method: "POST",
        })
        const result = await response.json()

        if (result.code === "0") {
            alert("当前没有节点在线！")
            return
        }
        result.nodes.forEach(node => {
            const listItem = document.createElement("li")
            const nodeInfo = document.createElement("p")
            nodeInfo.textContent = node.nodeName + "<--->" + node.nodeAddr
            nodeInfo.style.margin = "5px 0"
            nodeInfo.style.padding = "10px"
            nodeInfo.style.border = "1px solid #ccc"
            nodeInfo.style.borderRadius = "5px"
            nodeInfo.style.backgroundColor = "#f8f9fa"
            nodeInfo.style.fontSize = "bold"
            listItem.appendChild(nodeInfo)
            nodes_exist_list.appendChild(listItem)
        })
    } catch (error) {
        console.error("节点查询失败：", error)

    }
})
