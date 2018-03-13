
function errorHandler(message) {
  $('.errormsg').stop().text(message).slideDown().delay(4000).fadeOut(500)
}

function noticeHandler(message) {
  $('.noticemsg').stop().text(message).slideDown().delay(4000).fadeOut(500)
}
