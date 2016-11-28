lib := mclib
repo := github.com/jdrivas/$(lib)

help:
	@echo check \# run a grep to look for awslib that haven't been commented.
	@echo release \# push master branch to github and then do a local go update.

check:
	@ if grep -e '^[[:space:]]*\"awslib\"' *go ; then \
		echo "Fix the library refrence."; \
		exit -1; \
	else echo "Checked o.k."; \
	fi


release: check
	@echo Pushing $(repo) to git and getting local copy of library to go env.
	git push
	go get -u $(repo)
