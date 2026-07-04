.PHONY: help verify meta-test test constitution-check

help: ## List available constitution-inheritance targets
	@echo "Available targets:"
	@echo "  verify              - Run the constitution-inheritance gate (tests/verify_constitution_inheritance.sh)"
	@echo "  meta-test           - Run the false-positive proof meta-test (scripts/testing/meta_test_false_positive_proof.sh)"
	@echo "  test                - Run the constitution-inheritance host test (tests/test_constitution_inheritance.sh)"
	@echo "  constitution-check  - Run verify, then meta-test, then test (fail-fast)"
	@echo "  help                - Show this message"

verify: ## Run the constitution-inheritance gate
	bash tests/verify_constitution_inheritance.sh

meta-test: ## Run the false-positive proof meta-test
	bash scripts/testing/meta_test_false_positive_proof.sh

test: ## Run the constitution-inheritance host test
	bash tests/test_constitution_inheritance.sh

constitution-check: verify meta-test test ## Run verify, meta-test, and test in order (fail-fast)
	@echo "constitution-check: all stages passed"
