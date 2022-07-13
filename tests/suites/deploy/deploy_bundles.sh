run_deploy_bundle() {
	echo

	file="${TEST_DIR}/test-bundles-deploy.log"

	ensure "test-bundles-deploy" "${file}"

	juju deploy juju-qa-bundle-test
	wait_for "juju-qa-test" ".applications | keys[0]"
	wait_for "ntp" "$(idle_subordinate_condition "ntp" "juju-qa-test")"

	destroy_model "test-bundles-deploy"
}

run_deploy_bundle_overlay() {
	echo

	file="${TEST_DIR}/test-bundles-deploy-overlay.log"

	ensure "test-bundles-deploy-overlay" "${file}"

	bundle=./tests/suites/deploy/bundles/overlay_bundle.yaml
	juju deploy ${bundle}

	wait_for "ubuntu" "$(idle_condition "ubuntu" 0 0)"
	wait_for "ubuntu" "$(idle_condition "ubuntu" 0 1)"

	destroy_model "test-bundles-deploy-overlay"
}

run_deploy_cmr_bundle() {
	echo

	file="${TEST_DIR}/test-cmr-bundles-deploy.log"

	ensure "test-cmr-bundles-deploy" "${file}"

	# mysql charm does not have stable channel, so we use edge channel
	juju deploy mysql --channel=edge
	wait_for "mysql" ".applications | keys[0]"

	juju offer mysql:db
	juju add-model other

	juju switch other

	bundle=./tests/suites/deploy/bundles/cmr_bundles_test_deploy.yaml
	sed "s/{{BOOTSTRAPPED_JUJU_CTRL_NAME}}/${BOOTSTRAPPED_JUJU_CTRL_NAME}/g" "${bundle}" >"${TEST_DIR}/cmr_bundles_test_deploy.yaml"
	# TODO - upgrade this bundle to use focal
	# Must wait until the wordpress charm is updated to support focal/jammy
	# https://charmhub.io/wordpress
	juju deploy "${TEST_DIR}/cmr_bundles_test_deploy.yaml"

	wait_for "wordpress" "$(idle_condition "wordpress")"

	destroy_model "test-cmr-bundles-deploy"
	destroy_model "other"
}

run_deploy_exported_bundle() {
	echo

	file="${TEST_DIR}/test-export-bundles-deploy.log"

	ensure "test-export-bundles-deploy" "${file}"

	bundle=./tests/suites/deploy/bundles/telegraf_bundle.yaml
	juju deploy ${bundle}

	# no need to wait for the bundle to finish deploying to
	# check the export.
	juju export-bundle --filename "${TEST_DIR}/exported-bundle.yaml"
	diff ${bundle} "${TEST_DIR}/exported-bundle.yaml"

	destroy_model "test-export-bundles-deploy"
}

run_deploy_trusted_bundle() {
	echo

	file="${TEST_DIR}/test-trusted-bundles-deploy.log"

	ensure "test-trusted-bundles-deploy" "${file}"

	# TODO - upgrade the charm to support focal
	bundle=./tests/suites/deploy/bundles/trusted_bundle.yaml
	OUT=$(juju deploy ${bundle} 2>&1 || true)
	echo "${OUT}" | check "repeat the deploy command with the --trust argument"

	juju deploy --trust ${bundle}

	wait_for "trust-checker" "$(idle_condition "trust-checker")"

	destroy_model "test-trusted-bundles-deploy"
}

run_deploy_charmhub_bundle() {
	echo

	model_name="test-charmhub-bundle-deploy"
	file="${TEST_DIR}/${model_name}.log"

	ensure "${model_name}" "${file}"

	bundle=juju-qa-bundle-test
	juju deploy "${bundle}"

	wait_for "juju-qa-test" "$(charm_channel "juju-qa-test" "2.0/stable")"
	wait_for "juju-qa-test-focal" "$(charm_channel "juju-qa-test-focal" "candidate")"
	wait_for "juju-qa-test" "$(idle_condition "juju-qa-test")"
	wait_for "juju-qa-test-focal" "$(idle_condition "juju-qa-test-focal" 1)"
	wait_for "ntp" "$(idle_subordinate_condition "ntp" "juju-qa-test")"
	wait_for "ntp-focal" "$(idle_subordinate_condition "ntp-focal" "juju-qa-test-focal")"

	destroy_model "${model_name}"
}

# run_deploy_lxd_profile_bundle_openstack is to test a more
# real world scenario of a minimal openstack bundle with a
# charm using an lxd profile.
run_deploy_lxd_profile_bundle_openstack() {
	echo

	model_name="test-deploy-lxd-profile-bundle-o7k"
	file="${TEST_DIR}/${model_name}.log"

	ensure "${model_name}" "${file}"

	bundle=./tests/suites/deploy/bundles/basic-openstack.yaml
	juju deploy "${bundle}"

	wait_for "mysql" "$(idle_condition "mysql" 2)"
	wait_for "rabbitmq-server" "$(idle_condition "rabbitmq-server" 8)"
	wait_for "glance" "$(idle_condition "glance" 0)"
	wait_for "keystone" "$(idle_condition "keystone" 1)"
	wait_for "neutron-api" "$(idle_condition "neutron-api" 3)"
	wait_for "neutron-gateway" "$(idle_condition "neutron-gateway" 4)"
	wait_for "nova-compute" "$(idle_condition "nova-compute" 7)"
	wait_for "neutron-openvswitch" "$(idle_subordinate_condition "neutron-openvswitch" "nova-compute")"
	wait_for "nova-cloud-controller" "$(idle_condition "nova-cloud-controller" 6)"

	lxd_profile_name="juju-${model_name}-neutron-openvswitch"
	machine_6="$(machine_path 6)"
	juju status --format=json | jq "${machine_6}" | check "${lxd_profile_name}"

	destroy_model "${model_name}"
}

# run_deploy_lxd_profile_bundle is to deploy multiple units of the
# same charm which has an lxdprofile in a bundle.  The scenario
# created by the bundle was found to produce failure cases during
# development of the lxd profile feature.
run_deploy_lxd_profile_bundle() {
	echo

	model_name="test-deploy-lxd-profile-bundle"
	file="${TEST_DIR}/${model_name}.log"

	ensure "${model_name}" "${file}"

	bundle=./tests/suites/deploy/bundles/lxd-profile-bundle.yaml
	# TODO - upgrade the charm to support focal
	juju deploy "${bundle}"

	# 8 units of lxd-profile
	for i in 0 1 2 3 4 5 6 7; do
		wait_for "lxd-profile" "$(idle_condition "lxd-profile" 0 "${i}")"
	done
	# 4 units of ubuntu
	for i in 0 1 2 3; do
		wait_for "ubuntu" "$(idle_condition "ubuntu" 1 "${i}")"
	done

	lxd_profile_name="juju-${model_name}-lxd-profile"
	for i in 0 1 2 3; do
		machine_n_lxd0="$(machine_container_path "${i}" "${i}"/lxd/0)"
		juju status --format=json | jq "${machine_n_lxd0}" | check "${lxd_profile_name}"
		machine_n_lxd1="$(machine_container_path "${i}" "${i}"/lxd/1)"
		juju status --format=json | jq "${machine_n_lxd1}" | check "${lxd_profile_name}"
	done

	destroy_model "${model_name}"
}

test_deploy_bundles() {
	if [ "$(skip 'test_deploy_bundles')" ]; then
		echo "==> TEST SKIPPED: deploy bundles"
		return
	fi

	(
		set_verbosity

		cd .. || exit

		run "run_deploy_bundle"
		run "run_deploy_bundle_overlay"
		run "run_deploy_exported_bundle"
		run "run_deploy_trusted_bundle"
		run "run_deploy_charmhub_bundle"

		case "${BOOTSTRAP_PROVIDER:-}" in
		"lxd" | "localhost")
			run "run_deploy_lxd_profile_bundle_openstack"
			run "run_deploy_lxd_profile_bundle"
			;;
		*)
			echo "==> TEST SKIPPED: deploy_lxd_profile_bundle_openstack - tests for LXD only"
			echo "==> TEST SKIPPED: deploy_lxd_profile_bundle - tests for LXD only"
			;;
		esac

		# Run this last so the other tests run, there are intermittent issues
		# in cmr tear down.
		run "run_deploy_cmr_bundle"
	)
}
