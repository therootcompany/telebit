console.log("app.sh startup")


var app = angular.module("rvpnApp", ["ngRoute"]);
app.config(function($routeProvider, $locationProvider) {
    $routeProvider
    .when("/admin/servers", {
        templateUrl : "admin/partials/servers.html"
    })
    .when("/admin/domains", {
        templateUrl : "green.htm"
    })
    .when("/blue", {
        templateUrl : "blue.htm"
    });
    $locationProvider.html5Mode(true);
});