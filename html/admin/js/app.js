console.log("app.sh startup");

var app = angular.module("rvpnApp", ["ngRoute", "angular-duration-format"]);

app.config(function ($routeProvider, $locationProvider) {
    $routeProvider

        .when("/admin/status/", {
            templateUrl: "admin/partials/status.html"
        })

        .when("/admin/index.html", {
            templateUrl: "admin/partials/servers.html"
        })

        .when("/admin/servers/", {
            templateUrl: "admin/partials/servers.html"
        })

        .when("/admin/#domains", {
            templateUrl: "green.htm"
        })

        .when("/blue", {
            templateUrl: "blue.htm"
        });
    $locationProvider.html5Mode(true);
});

app.filter("bytes", function () {
    return function (bytes, precision) {
        if (isNaN(parseFloat(bytes)) || !isFinite(bytes)) return "-";
        if (typeof precision === "undefined") precision = 1;
        var units = ["bytes", "kB", "MB", "GB", "TB", "PB"],
            number = Math.floor(Math.log(bytes) / Math.log(1024));
        return (bytes / Math.pow(1024, Math.floor(number))).toFixed(precision) + " " + units[number];
    };
});

app.filter("hfcduration", function () {
    return function (duration, precision) {
        remain = duration;
        duration_day = 24 * 60 * 60;
        duration_hour = 60 * 60;
        duration_minute = 60;
        duration_str = "";

        days = Math.floor(remain / duration_day);
        if (days > 0) {
            remain = remain - days * duration_day;
            duration_str = duration_str + days + "d";
        }

        hours = Math.floor(remain / duration_hour);
        if (hours > 0) {
            remain = remain - hours * duration_hour;
            duration_str = duration_str + hours + "h";
        }

        mins = Math.floor(remain / duration_minute);
        if (mins > 0) {
            remain = remain - mins * duration_minute;
            duration_str = duration_str + mins + "m";
        }

        secs = Math.floor(remain);
        duration_str = duration_str + secs + "s";

        return duration_str;
    };
});

app.controller("statusController", function ($scope, $http) {
    console.log("statusController");
    $scope.status_search = "";

    var api = "/api/org.rootprojects.tunnel/status";

    $scope.updateView = function () {
        $http.get(api).then(function (response) {
            console.log(response);
            data = response.data;
            if (data.error == "ok") {
                $scope.status = data.result;
            }
        });
    };

    $scope.updateView();
});

app.controller("serverController", function ($scope, $http) {
    $scope.servers = [];
    $scope.servers_search = "";
    $scope.servers_trigger_details = [];
    $scope.filtered;

    var api = "/api/org.rootprojects.tunnel/servers";

    $scope.updateView = function () {
        $http.get(api).then(function (response) {
            //console.log(response);
            data = response.data;
            if (data.error == "ok") {
                $scope.servers = data.result.servers;
            }
        });
    };

    $scope.triggerDetail = function (id) {
        //console.log("triggerDetail ", id, $scope.servers_trigger_details[id])
        if ($scope.servers_trigger_details[id] == true) {
            $scope.servers_trigger_details[id] = false;
        } else {
            $scope.servers_trigger_details[id] = true;
        }
    };

    $scope.checkDetail = function (id) {
        //console.log("checkDetail ", id, $scope.servers_trigger_details[id])
        if ($scope.servers_trigger_details[id] == true) {
            return false;
        } else {
            return true;
        }
    };

    $scope.updateView();
});
