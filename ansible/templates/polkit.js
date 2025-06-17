polkit.addRule(function(action, subject) {
	if (!subject.isInGroup("{{ flipcam_group }}")) {
		return polkit.Result.NOT_HANDLED;
	}

	if (action.id !== "org.freedesktop.systemd1.manage-units") {
		return polkit.Result.NOT_HANDLED;
	}

	var unit = action.lookup("unit")
	if (unit !== "{{ caddy_service_name }}"
		&& unit !== "{{ dnsmasq_service_name }}"
		&& unit !== "{{ hostapd_service_name }}") {
		return polkit.Result.NOT_HANDLED;
	}

	var actionVerb = action.lookup("verb")
	if (actionVerb !== "start"
		&& actionVerb !== "stop") {
		return polkit.Result.NOT_HANDLED;
	}

	return polkit.Result.YES;
});
