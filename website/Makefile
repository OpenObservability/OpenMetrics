get-sass:
	yarn

netlify-production-build: get-sass
	hugo

netlify-staging-build: get-sass
	hugo -b $(DEPLOY_PRIME_URL)

dev:
	hugo server --disableFastRender
