.PHONY: debug-login demo

# 自动加载 backend/.env 中的环境变量
ifneq (,$(wildcard backend/.env))
    include backend/.env
    export
endif

debug-login:
	./scripts/dev-login.sh $(UID)

demo:
	cd backend && make demo