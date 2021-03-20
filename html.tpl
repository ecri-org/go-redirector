<html>
<head>
	<meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
</head>
<body>
<p>The page you reached has moved to <a href="{{.RedirectUri}}">{{.RedirectUri}}</a>, please update your bookmarks.</p>
<p>You will be automatically redirected to {{.RedirectUri}} in <span id="countdown">15</span> seconds.</p>
<p>Or click <a href="{{.RedirectUri}}">THIS LINK</a> to go there now.</p>
<script type="text/javascript">
	let seconds = 15;

	function countdown() {
		seconds = seconds - 1;
		if (seconds < 0) {
			window.location = "{{.RedirectUri}}";
		} else {
			document.getElementById("countdown").innerHTML = seconds.toString();
			window.setTimeout("countdown()", 1000);
		}
	}
	countdown();
</script>
<p hidden>Generated from a simple-redirector template.</p>
</body>
</html>