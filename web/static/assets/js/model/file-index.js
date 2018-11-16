var centerUrl = "39.104.112.203:8080";
var niming = 0; //0 是匿名，1 是用户
var currentdir = 0;
var currentpage = 1;
var currentkind = "*";
var currentDelete = 0;
var loginuser = 1;
var iswindows = false;
function init() {
    var url = location.search; //获取url中"?"符后的字串 
    // var theRequest = new Object();

    if (url.indexOf("?") != -1) {
        var str = url.substr(1);
        strs = str.split("&");
        for (var i = 0; i < strs.length; i++) {
            if (strs[i].split("=")[0] == "user") {
                loginuser = unescape(strs[i].split("=")[1]);
                break;
            }
            // theRequest[strs[i].split("=")[0]] = unescape(strs[i].split("=")[1]);
        }
    }
    if (loginuser == 0) {
        var sessionid = getCookie("Sessionid");
        if (sessionid == null || sessionid == "") {
            getAnonymousFileList("*");
        } else {
            doLogoutFunc("?user=0");
        }
    } else if (loginuser == 1) {
        myMessage();
        getFileList(currentpage, currentkind, currentDelete, 0);
    }


    if (navigator.userAgent.indexOf(".NET") >= 0) {
        // $("#downloadBT").hide(); 
        iswindows = true;
    }
}

function openFileList(kind, deleteit, parent_id, buttonItem) {
    var collapseNav = document.getElementsByClassName("menuLi");
    for (var i = 0; i < collapseNav.length; i++) {
        collapseNav[i].style.backgroundColor = "#343a3d";
    }
    buttonItem.style.backgroundColor = "#242b2e";
    currentDelete = deleteit;
    if (loginuser == 0) {
        getAnonymousFileList(kind);
    } else {
        $("#rubbishUI").show();
        $("#newdirBT").show();
        $("#deleteBT").show();

        getFileList(1, kind, currentDelete, parent_id);
    }

}

function getFileList(pagenum, kind, del, parent_id) {
    $("#userinfoBar").show();
    $("#nimingBar").hide();
    if (pagenum <= 0) {
        return
    }
    if (parent_id <= 0) {
        parent_id = 0;//currentdir;
    }


    if (del == 1) {
        $("#uploadBT").hide();
        $("#downloadBT").hide();
        $("#newdirBT").hide();
        $("#deleteBT").text("恢复选择");
        $("#removeBT").show();

    } else {
        $("#uploadBT").show();
        $("#downloadBT").show();
        $("#newdirBT").show();
        $("#deleteBT").text("移除选择");
        $("#removeBT").hide();
    }
    currentDelete = del;

    var sessionid = getCookie("Sessionid");
    $.ajax({
        headers: {
            Sessionid: sessionid
        },
        url: centerUrl + '/api/file/mine',
        type: 'get',
        data: {
            // limit: "10",
            // offset: 10 * (pagenum - 1),
            kind: kind,
            delete: del,
            parentid: parent_id,
        },
        dataType: 'json',
        success: function (res) {
            if (res.status != 200) {
                $(window.location).attr('href', './web-login.html');
            } else {
                if (res.status == 401) {
                    $(window.location).attr('href', './web-login.html');
                    return
                }
                document.getElementById("quanxuanCB").checked = false;
                $("#mainTable").empty();
                currentpage = pagenum;
                currentkind = kind;
                currentDelete = del;
                currentdir = parent_id;
                $("#currentPage").text(pagenum);
                if (res.data == null) {
                    //addClass("am-disabled")
                    return
                }


                res.data.forEach(function (item, index, input) {
                    displayItemFile(item, del);
                })
            }

        }
    })
}
var myObj = {};
function myMessage() {
    $("#messageContent").children().remove();
    $("#messageLength").text("");
    var sessionid = getCookie("Sessionid")
    $.ajax({
        headers: {
            Sessionid: sessionid
        },
        url: centerUrl + '/api/message/mine',
        type: 'GET',
        data: {
            readed: 0,
        },
        dataType: 'json',
        success: function (res) {
            if (res.status == 500) {
                console.log(res.msg);
                return
            }
            if (res.data == null) {
                console.log(res.msg);
                return
            }
            $.data(myObj, 'usermessage', res.data);
            k = 0;
            res.data.forEach(function (item, index, input) {
                k++
                displayMessage(item);
            })
            $("#messageLength").text(k);

        }
    })
};

function displayMessage(item) {
    var time = new Date();
    var accountstr = '   <li>  <a href="#" onclick="openMessage(' + item.id + ')">' + item.title + '</a>  </li>';
    $("#messageContent").append(accountstr);
}


function openMessage(id) {
    $("#messageContent").children().remove();
    $("#messageLength").text("");
    var messagelist = $.data(myObj, 'usermessage');
    for (var i = 0; i < messagelist.length; i++) {
        if (messagelist[i].id != id) {
            continue;
        }
        var panel = '   <div class="hw-overlay" id="message-layer" style="display:none;z-index:99999;">' +
            '<div class="hw-layer-wrap" style=" width:450px;" >' +
            '  <h3 style="margin:-50px;border: 2px solid #1999e3;  padding-left:20px; width:auto;   color: white; ' +
            ' font-family: MicrosoftYaHei;  font-size: 24px;  font-weight: normal;  font-stretch: normal;    height: 46px;' +
            '  background-color: #1999e3; ">消息 <label   style=" cursor:  pointer;color:#fff; margin-left:300px; " onclick="cancelMessagepanel()">╳</label> </h3 >' +
            ' <br>   <br>    <div class="row">' +

            '  <form class="am-form" method="POST"> ' +
            ' <div class="am-g am-margin-top"> ' +
            ' </div>' +
            '  <div class="am-g am-margin-top">' +
            '   <div class="am-u-sm-2 am-u-md-3  am-text-right">    标题:' +
            ' </div>' +
            ' <div class="am-u-sm-10 am-u-md-9  ">' +
            ' <label>' + messagelist[i].title + '  </label>' +
            '  </div>' +
            ' </div>' +

            ' <div class="am-g am-margin-top">' +
            ' <div class="am-u-sm-2 am-u-md-3  am-text-right">  内容:' +
            ' </div>' +
            '<div class="am-u-sm-10 am-u-md-9  ">' +
            '<label>' + messagelist[i].content + '  </label>' +
            '</div>' +
            ' </div>  ' +

            '</form>   <hr> ' +
            ' <button type="reset" class="am-btn am-btn-primary am-btn-xs am-fr  hwLayer-cancel" onclick="cancelMessagepanel()">确  定</button>' +
            ' </div>' +
            ' </div>' +
            ' </div>';

        $("#mainBody").prepend(panel);
        $("#message-layer").show();
        var topnum = 10;
        $("#message-layer").find(".hw-layer-wrap").css("margin-top", topnum);
        var sessionid = getCookie("Sessionid");
        $.ajax({
            url: centerUrl + '/api/message/readed/' + id,
            type: 'GET',
            headers: {
                Sessionid: sessionid
            },
            dataType: 'json',
            success: function (res) {
                if (res.status == 500) {
                    return
                }
                if (res.data == null) {
                    return
                }
                k = 0;
                res.data.forEach(function (item, index, input) {
                    k++
                    displayMessage(item);
                })
                $("#messageLength").text(k);

            }
        })



        break;
    }
}
function cancelMessagepanel() {
    $("#message-layer").hide();
    $("#message-layer").remove();
}
function newDirctionary() {
    var panel = ' <div class="hw-overlay" id="dirction-layer" style="display:none;z-index:99999;">' +
        ' <div class="hw-layer-wrap" style=" width:450px;" >' +
        ' <h3 style="margin:-50px;border: 2px solid #1999e3;' +
        ' padding-left:20px; width:auto;' +
        ' color: white; ' +
        ' font-family: MicrosoftYaHei;' +
        ' font-size: 24px;' +
        ' font-weight: normal;' +
        ' font-stretch: normal;' +
        ' height: 46px;' +
        ' background-color: #1999e3; "> 新建文件夹  <label   style=" cursor:  pointer;color:#fff; margin-left:270px; " onclick="cancelDirctionpanel()">╳</label> </h3 >' +
        ' <br>' +
        '  <br>' +
        '  <div class="row" style="margin-top:50px;">' +

        '   <div class="am-g am-margin-top">' +

        '    <label><input id="newdir" type="text" placeholder="文件夹名称"  style="width:350px;height:46px;padding-left:10px;" />  </label>' +

        '    <br>' +
        '    <br>' +
        '    <label   style=" cursor:  pointer;color:#1999e3;padding-left:20px;" onclick="createNewDir(newdir)" class=" hwLayer-ok">确 定' +
        '    </label>' +
        '    <label   style=" cursor:  pointer;color:#1999e3;padding-left:250px;"   class="  hwLayer-cancel" onclick="cancelDirctionpanel()">取 消</label>' +

        '  </div>' +
        '  </div>' +
        ' </div>';

    $("#mainBody").prepend(panel);
    $("#dirction-layer").show();
    var topnum = 10;
    $("#dirction-layer").find(".hw-layer-wrap").css("margin-top", topnum);
}
function createNewDir(newdirItem) {
    var dirItem = {};
    dirItem.title = newdirItem.value;
    dirItem.kind = "dir";
    dirItem.ext = "dir";
    dirItem.delete = 0;
    dirItem.parent_id = currentdir;
    var sessionid = getCookie("Sessionid")
    $.ajax({
        headers: {
            Sessionid: sessionid
        },
        url: centerUrl + '/api/file/createDir',
        type: 'post',
        data: JSON.stringify(dirItem),
        dataType: 'json',
        success: function (res) {
            console.log(currentdir);
            if (res.status != 200) {
                alert(res.msg);
            } else {
                if (res.status == 401) {
                    $(window.location).attr('href', './login.html');

                    return
                }
                if (res.data == null) {

                    return
                }
                displayItemFile(res.data);
                cancelDirctionpanel();
            }

        }
    })

}
function cancelDirctionpanel() {
    $("#dirction-layer").hide();
    $("#dirction-layer").remove();
}


function displayItemFile(item, deletitem) {
    var time = new Date();
    var typeImg = "assets/web/doc.png";
    var itemsize = item.size;
    var sizeresult = ""
    while (itemsize > 1000) {
        var itemsizeStr = itemsize.toString();
        if (sizeresult == "") {
            sizeresult = itemsizeStr.substring(itemsizeStr.length - 3, itemsizeStr.length);
        } else {
            sizeresult = itemsizeStr.substring(itemsizeStr.length - 3, itemsizeStr.length) + "," + sizeresult
        }

        itemsize = parseInt(itemsize / 1000);
    }
    if (itemsize == "0") {
        sizeresult = "0";
    } else {
        if (sizeresult == "") {
            sizeresult = itemsize.toString();
        } else {

            sizeresult = itemsize.toString() + "," + sizeresult;
        }
    }

    var controlBt = '<a target="_blank" download="' + item.title + '" href="\/store\/' + item.hash_code + '"" style="padding-top:15px;" class="am-btn am-btn-default am-btn-xs am-hide-sm-only"><span class="am-icon-save"></span> </a>';
    var hrefStr = '<a href="#"  onclick="downloadThisFile(\'' + item.hash_code + '\')">' + item.title + '</a>';
    if (item.kind == "dir") {
        hrefStr = '<a href="#"  onclick="openThisFile(\'' + item.id + '\' )">' + item.title + '</a>';
    }
    var blankStr = 'target="_blank"  download="' + item.title + '" ';
    if (iswindows == false) {
        blankStr = 'target="_blank"  ';
    }
    switch (item.kind) {
        case "pic":
            typeImg = "assets/web/tp.png";
            hrefStr = ' <a ' + blankStr + ' id="' + item.hash_code + '" href="\/store\/' + item.hash_code + '"    > ' + item.title + '</a>';
            break;
        case "video":
            typeImg = "assets/web/sp.png";
            hrefStr = ' <a ' + blankStr + '   id="' + item.hash_code + '" href="\/store\/' + item.hash_code + '"   > ' + item.title + '</a>';
            break;
        case "doc":
            typeImg = "assets/web/doc.png";
            hrefStr = ' <a ' + blankStr + ' id="' + item.hash_code + '" href="\/store\/' + item.hash_code + '"   > ' + item.title + '</a>';
            break;
        case "other":
            typeImg = "assets/web/wgx.png";
            hrefStr = ' <a ' + blankStr + '   id="' + item.hash_code + '" href="\/store\/' + item.hash_code + '"  > ' + item.title + '</a>';
            break;
        case "dir":
            typeImg = "assets/web/dir.png";
            // hrefStr = ' <a download="' + item.title + '" href="\/store\/' + item.hash_code + '""  > ' + item.title + '</a>'; 
            controlBt = "";
            sizeresult = "";
            break;
        default:
            typeImg = "assets/web/wgx.png";
            hrefStr = ' <a ' + blankStr + '  id="' + item.hash_code + '" href="\/store\/' + item.hash_code + '"   > ' + item.title + '</a>';
            break;
    }
    var actionStr = '  <button onclick="killItem(' + item.id + ')" class="am-btn am-btn-default am-btn-xs am-text-danger am-hide-sm-only"><span class="am-icon-trash-o"></span>  </button>';
    if (deletitem == 1) {
        controlBt = ' <button  onclick="reuseItem(' + item.id + ')" class="am-btn am-btn-default am-btn-xs am-hide-sm-only"><img src="assets/web/hf.png"  style="width: 20px;height:22px;" />   恢复</button>';
        actionStr = '  <button onclick="killItem(' + item.id + ')" class="am-btn am-btn-default am-btn-xs am-text-danger am-hide-sm-only"><img src="assets/web/yc.png"  style="width: 20px;height:22px;" />  彻底移除</button>';
    }
    var accountstr = '   <tr>' +
        '<td><input type="checkbox" placeholder="' + item.title + '" name="' + item.hash_code + '" value="' + item.id + '" class="qx"  /></td>' +

        '<td> <img style=" width:32px;height:32px;" src="' + typeImg + '"  />&nbsp;&nbsp;' + hrefStr + '</td>' +
        '<td>' + sizeresult + '</td>' +
        ' <td >' + item.hash_code + '</td>' +
        ' <td class="am-hide-sm-only">' + item.upload_time + '</td>' +
        ' <td>' +
        '  <div class="am-btn-toolbar" style="text-align:right;">' +
        '    <div class="am-btn-group am-btn-group-xs">' +

        '     ' + controlBt + actionStr + '' +

        '    </div>' +
        '  </div>' +
        ' </td>' +
        '</tr>';
    $("#mainTable").append(accountstr);
}
function openThisFile(id) {
    getFileList(currentpage, currentkind, currentDelete, id);

}
function downloadThisFile(hash_code) {
    var href = "/store/" + hash_code;
    window.open(href, 'top');
}

function download(src, title, hash) {
    var thisa = document.getElementById(hash);
    var evt = document.createEvent("MouseEvent");
    evt.initMouseEvent("click", true, true, window, 0, 0, 0, 80, 20, false, false, false, false, 0, null);
    thisa.dispatchEvent(evt);

};


function batchDownload() {
    var hashlist = getHashQuanxuanList();

    if (hashlist.length > 0) {
        hashlist.forEach(function (item, index, input) {
            download("/store/" + item["hash"], item["title"], item["hash"]);
        });
    } else {
        alert("没有选择");
    }
}

function backparentdir() {
    if (currentdir == "0") {
        alert("已经是最顶级");
        return
    }
    var sessionid = getCookie("Sessionid")
    $.ajax({
        headers: {
            Sessionid: sessionid
        },
        url: centerUrl + '/api/file/backParent/' + currentdir,
        type: 'get',
        data: {
            // limit: "10",
            // offset: 0,
            delete: currentDelete,
        },
        dataType: 'json',
        success: function (res) {
            if (res.status != 200) {
                alert(res.msg);
            } else {
                if (res.status == 401) {
                    $(window.location).attr('href', './login.html');
                    return
                }

                if (res.msg != "top") {
                    $("#mainTable").empty();
                    $("#currentPage").text(0);
                    currentdir = res.msg;
                    if (res.data == null) {
                        return;
                    }
                    res.data.forEach(function (item, index, input) {
                        displayItemFile(item);
                    })
                } else {
                    alert("已经是最顶级");
                }
            }

        }
    })


}
function reuseItem(idlist) {
    var r = confirm("真的要恢复?")
    if (r == true) {
        currentDelete = 1;
        dokillItem(idlist, 0);
    }

}
function killItem(id) {
    var r = confirm("真的要删?")
    if (r == true) {
        dokillItem(id, 1);
    }

}
function dokillItem(idlist, kill) {
    if (idlist.length == 0) {
        alert("没有选择");
        return
    }
    var sessionid = getCookie("Sessionid")
    $.ajax({
        headers: {
            Sessionid: sessionid
        },
        url: centerUrl + '/api/file/kill',
        type: 'get',
        data: {
            delete: kill,
            idList: idlist,
        },
        dataType: 'json',
        success: function (res) {
            console.log(" ");
            if (res.status != 200) {
                alert(res.msg);
            } else {
                if (res.status == 401) {
                    $(window.location).attr('href', './login.html');
                    return
                }
                getFileList(currentpage, currentkind, currentDelete, currentdir);
            }

        }
    })
}

function quanxuan(a) {
    //找到下面所有的复选框
    var ck = document.getElementsByClassName("qx");
    //遍历所有复选框，设置选中状态。
    for (var i = 0; i < ck.length; i++) {
        if (a.checked)//判断全选按钮的状态是不是选中的
        {
            ck[i].setAttribute("checked", "checked");//如果是选中的，就让所有的状态为选中。
        }
        else {
            ck[i].removeAttribute("checked");//如果不是选中的，就移除所有的状态是checked的选项。
        }
    }
}

function removeAll() {
    var r = confirm("真的要彻底移除选择?")
    if (r == true) {
        var idlist = getQuanxuanList()
        dokillItem(idlist, -1);
    }
}

function deleteAll() {
    if (currentDelete == "1") {
        var r = confirm("真的要恢复选择?")
        if (r == true) {
            var idlist = getQuanxuanList()
            dokillItem(idlist, 0);
        }
    } else {
        var r = confirm("真的要移除选择?")
        if (r == true) {
            var idlist = getQuanxuanList()
            dokillItem(idlist, 1);
        }
    }

}
function getHashQuanxuanList() {
    var ck = document.getElementsByClassName("qx");
    var hashlist = new Array();
    var j = 0;
    //遍历所有复选框，设置选中状态。
    for (var i = 0; i < ck.length; i++) {
        if (ck[i].checked) {
            var item = {};
            item["hash"] = ck[i].name;
            item["title"] = ck[i].placeholder;
            hashlist[j] = item
            j++;
        }
    }
    return hashlist
}
function getQuanxuanList() {
    var ck = document.getElementsByClassName("qx");
    var idlist = "";
    //遍历所有复选框，设置选中状态。
    for (var i = 0; i < ck.length; i++) {
        if (ck[i].checked) {
            if (idlist == "") {
                idlist = ck[i].value;
            } else {
                idlist = idlist + "," + ck[i].value;
            }
        }
    }
    return idlist
}

function uploadfile() {
    var panel = ' <div class="hw-overlay" id="uploadfile-layer" style="display:none;z-index:99999;">' +
        ' <div class="hw-layer-wrap" style=" width:450px;" >' +
        '   <h3 style="margin:-50px;border: 2px solid #1999e3;' +
        '   padding-left:20px; width:auto; height: 46px;' +
        '  font-family: MicrosoftYaHei; color: white;  font-size: 24px;' +
        '  font-weight: normal;  font-stretch: normal; background-color: #1999e3; ' +
        '  ">上传文件 <label   style=" cursor:  pointer;color:#fff; margin-left:270px; " onclick="cancelUploadpanel()">╳</label> </h3 >' +
        '  <br>   <br>   <div class="row" style="margin-top:50px;">' +
        '  <div class="am-g am-margin-top">' +
        '  <form id="fileForm" enctype="multipart/form-data">' +
        '    <label><input id="fileInput" type="file"   style="width:350px;height:46px;padding-left:10px;" />  </label>' +
        '       <br>   <br>    <label   style=" cursor:  pointer;color:#1999e3;padding-left:20px;" onclick="doUploadfile(fileInput)" class=" hwLayer-ok">确定 </label>' +
        '     <label   style=" cursor:  pointer;color:#1999e3;padding-left:250px;"   class="  hwLayer-cancel" onclick="cancelUploadpanel()">取 消</label>' +
        '     </form>' +
        '   </div>' +
        ' </div>' +
        '</div>';
    $("#mainBody").prepend(panel);
    $("#uploadfile-layer").show();
    var topnum = 10;
    $("#uploadfile-layer").find(".hw-layer-wrap").css("margin-top", topnum);
}
//根据扩展名 返回文件的类型。文件类型包括  pic图片，video视频，doc文档,dir目录，other其他，
function getFileKind(extStr) {
    var fileKind = "other";
    switch (extStr) {
        case "jpg":
        case "png":
        case "PNG":
        case "JPG":
        case "GIF":
        case "jpeg":
        case "gif":
            fileKind = "pic";
            break;
        case "avi":
        case "mpeg":
            fileKind = "video";
            break;
        case "doc":
        case "docx":
        case "pdf":
        case "xls":
            fileKind = "doc";
            break;
    }
    return fileKind;
}
function getFileExt(filename) {
    var i = filename.lastIndexOf("\\");
    var tmp_imgname = filename.slice(i + 1);
    var fileName = tmp_imgname.lastIndexOf(".");//取到文件名开始到最后一个点的长度
    var fileNameLength = tmp_imgname.length;//取到文件名长度
    return tmp_imgname.substring(fileName + 1, fileNameLength);//截
}
function doUploadfile(fileinput) {
    var formData = new FormData();
    formData.append('files[]', $('#fileInput')[0].files[0]);
    var filefullpath = fileinput.value;
    var i = filefullpath.lastIndexOf("\\");
    var tmp_imgname = filefullpath.slice(i + 1);
    var fileFormat = getFileExt(filefullpath);
    var fileKind = "other";
    fileKind = getFileKind(fileFormat);
    var sessionid = getCookie("Sessionid")
    $.ajax({
        headers: {
            Sessionid: sessionid
        },
        url: '/store/addfile',
        type: 'post',
        data: formData,
        dataType: 'json',
        processData: false,
        contentType: false,
        cache: false,
        success: function (res) {
            if (res.Code != 0) {
                alert(res.msg);
            } else {
                cancelUploadpanel();
                if (loginuser == 0) {
                    getAnonymousFileList(fileKind);
                } else if (loginuser == 1) {

                    var dirItem = {};
                    dirItem.title = tmp_imgname;
                    dirItem.kind = fileKind;
                    dirItem.ext = fileFormat;
                    dirItem.delete = 0;
                    dirItem.parent_id = parseInt(currentdir);
                    dirItem.hash_code = res.HashName;
                    dirItem.size = res.Size;
                    var sessionid = getCookie("Sessionid");
                    $.ajax({
                        headers: {
                            Sessionid: sessionid,
                        },
                        url: centerUrl + '/api/file/createDir',
                        type: 'post',
                        data: JSON.stringify(dirItem),
                        dataType: 'json',
                        processData: false,
                        contentType: false,
                        cache: false,
                        success: function (res) {
                            if (res.status != 200) {
                                alert(res.msg);
                            } else {

                                getFileList(currentpage, currentkind, currentDelete, currentdir);
                            }

                        }
                    })
                }


            }

        }
    })
}
function cancelUploadpanel() {
    $("#uploadfile-layer").hide();
    $("#uploadfile-layer").remove();
}
//-----------------------匿名模式------------------------
function useAnonymous() {
    var sessionid = getCookie("Sessionid");
    if (sessionid == null || sessionid == "") {
        getAnonymousFileList("");
        return;
    }
    //如果还有session  那么先注销session
    $.ajax({
        headers: {
            Sessionid: sessionid
        },
        url: centerUrl + '/api/user/logout',
        type: 'GET',
        dataType: 'json',
        success: function (res) {
            if (res.status != 200) {
                return
            }
            window.location.href = "?user=0";
        }
    })

}
function getAnonymousFileList(kindstr) {
    $("#userinfoBar").hide();
    $("#nimingBar").show();
    loginuser = 0;

    $("#mainTable").empty();
    $("#currentPage").text(0);
    $("#rubbishUI").hide();
    $("#newdirBT").hide();
    $("#deleteBT").hide();
    $.ajax({
        url: '/store/getlist',
        data: {
            kind: kindstr,
        },
        type: 'GET',
        dataType: 'json',
        success: function (res) {
            if (res.status != 200) {
                alert("匿名模式出错");
                return
            }
            if (res.data == null) {
                return;
            }
            if (kindstr == "") {
                kindstr = "*";
            }
            res.data.forEach(function (item, index, input) {
                var extStr = getFileExt(item.Name);
                var kindvalue = getFileKind(extStr);
                if (kindvalue == kindstr || kindstr == "*") {
                    var nimingItem = {};
                    nimingItem.kind = kindvalue;
                    nimingItem.title = item.Name;
                    nimingItem.delete = 0;
                    nimingItem.parent_id = 0;
                    nimingItem.hash_code = item.HasCode;
                    nimingItem.size = item.Size;
                    var date = new Date(item.Time * 1000);
                    nimingItem.upload_time = date.getFullYear() + "年" + (date.getMonth() + 1) + "月" + date.getDate() + '日 ' + date.getHours() + ':' + date.getMinutes();
                    displayItemFile(nimingItem);
                }
            })
        }
    })

}


//获取cookie
function getCookie(name) {
    var reg = RegExp(name + '=([^;]+)');
    var arr = document.cookie.match(reg);
    if (arr) {
        return arr[1];
    } else {
        return '';
    }
};