run_unregister() {
	echo

	file="${TEST_DIR}/unregister.log"

	ensure "unregister" "${file}"

	echo "Get controller name"
	controller_name=$(juju controllers --format=json | jq -r '."current-controller"')

	echo "Check controller is known"
	juju controllers --format=json | jq -r ".\"controllers\" | has(\"${controller_name}\")" | check true

	echo "Backup controller info before unregister"
	if [[ -f "${HOME}/.local/share/juju/controllers.yaml" ]]; then
		cp "${HOME}/.local/share/juju/controllers.yaml" "${HOME}/.local/share/juju/controllers.yaml.bak"
	fi
	if [[ -f "${HOME}/.local/share/juju/accounts.yaml" ]]; then
		cp "${HOME}/.local/share/juju/accounts.yaml" "${HOME}/.local/share/juju/accounts.yaml.bak"
	fi

	echo "Unregister controller"
	juju unregister --yes "${controller_name}"

	echo "Check controller is not known"
	juju controllers --format=json | jq -r ".\"controllers\".\"${controller_name}\"" | check null

	echo "Check the default controller is not equal to unregistered one"
	check_not_contains "$(juju controllers --format=json | jq -r '."current-controller"')" "${controller_name}"

	echo "Restore controller info after unregister"
	if [[ -f "${HOME}/.local/share/juju/controllers.yaml.bak" ]]; then
		mv "${HOME}/.local/share/juju/controllers.yaml.bak" "${HOME}/.local/share/juju/controllers.yaml"
	fi
	if [[ -f "${HOME}/.local/share/juju/accounts.yaml.ba"k ]]; then
		mv "${HOME}/.local/share/juju/accounts.yaml.bak" "${HOME}/.local/share/juju/accounts.yaml"
	fi

	juju controllers --format=json | jq -r '."current-controller"' | check "${controller_name}"

	destroy_model "unregister"
}

test_unregister() {
	if [ -n "$(skip 'test_unregister')" ]; then
		echo "==> SKIP: Asked to skip controller unregister tests"
		return
	fi

	(
		set_verbosity

		cd .. || exit

		run "run_unregister"
	)
}
