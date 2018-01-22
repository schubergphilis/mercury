
function errorHandler(text) {
  console.log("Error raised: "+ text)
}
/*
function getAPI(url) {
  var jqxhr = $.getJSON( url, function(data) {

    console.log( "success" + data );
    //if data.success == true {
      var jdata = JSON.parse(data.data)
      console.log("getApi data: " + jdata)
      return jdata
    //}
  })
    .done(function() {
      console.log( "second success" );
    })
    .fail(function() {
      console.log( "error" );
    })
    .always(function() {
      console.log( "complete" );
    });

}*/

/*
function getAPI(url) {
   var value= $.ajax({
      url: url,
      async: false
   }).responseJson;
   return value;
}
*/
