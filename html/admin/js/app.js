console.log("app.sh startup")

var app = angular.module("rvpnApp", ["ngRoute"]);
app.config(function($routeProvider, $locationProvider) {
    $routeProvider
    .when("/admin/index.html", {
        templateUrl : "admin/partials/servers.html"
    })
    .when("/admin/servers/", {
        templateUrl : "admin/partials/servers.html"
    })
    .when("/admin/#domains", {
        templateUrl : "green.htm"
    })
    .when("/blue", {
        templateUrl : "blue.htm"
    });
    $locationProvider.html5Mode(true);
});

app.controller('serverController', function ($scope, $http) {
    $scope.servers = [];
    var api = '/api/com.daplie.rvpn/servers'

    $http.get(api).then(function(response) {
        updateView(response.data);
    });

    updateView = function(data) {
        console.log(data);
        if (data.error == 'ok' ){
            console.log("ok")
            $scope.servers = data.result.servers;
            console.log(data.result)

        }
    };





});
    