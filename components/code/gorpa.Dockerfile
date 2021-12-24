# Copyright (c) 2018 Bhojpur Consulting Private Limited, India. All rights reserved.
# Licensed under the GNU Affero General Public License (AGPL).
# See License-AGPL.txt in the project root for license information.

# BUILDER_BASE is a placeholder, will be replaced before build time
# Check BUILD.yaml
FROM BUILDER_BASE as code_installer

ARG CODE_COMMIT

ENV PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD 1
ENV ELECTRON_SKIP_BINARY_DOWNLOAD 1

RUN mkdir bp-code \
    && cd bp-code \
    && git init \
    && git remote add origin https://github.com/bhojpur/vscode \
    && git fetch origin $CODE_COMMIT --depth=1 \
    && git reset --hard FETCH_HEAD
WORKDIR /bp-code
RUN yarn --frozen-lockfile --network-timeout 180000
RUN yarn --cwd ./extensions compile
RUN yarn gulp vscode-web-min
RUN yarn gulp vscode-reh-linux-x64-min

# config for first layer needed by blobserve
# we also remove `static/` from resource urls as that's needed by blobserve,
# this custom urls will be then replaced by blobserve.
# Check pkg/blobserve/blobserve.go, `inlineVars` method
RUN cp /vscode-web/out/vs/bhojpur/browser/workbench/workbench.html /vscode-web/index.html \
    && sed -i -e 's#static/##g' /vscode-web/index.html

# cli config: alises to bhojpur-code
# can't use relative symlink as they break when copied to the image below
COPY bin /ide/bin
RUN chmod -R ugo+x /ide/bin

# grant write permissions for built-in extensions
RUN chmod -R ugo+w /vscode-reh-linux-x64/extensions

FROM scratch
# copy static web resources in first layer to serve from blobserve
COPY --from=code_installer --chown=33333:33333 /vscode-web/ /ide/
COPY --from=code_installer --chown=33333:33333 /vscode-reh-linux-x64/ /ide/
COPY --chown=33333:33333 startup.sh supervisor-ide-config.json /ide/

COPY --from=code_installer --chown=33333:33333 /ide/bin /ide/bin

ENV BHOJPUR_ENV_APPEND_PATH /ide/bin:

# editor config
ENV BHOJPUR_ENV_SET_EDITOR /ide/bin/bhojpur-code
ENV BHOJPUR_ENV_SET_VISUAL "$BHOJPUR_ENV_SET_EDITOR"
ENV BHOJPUR_ENV_SET_BP_OPEN_EDITOR "$BHOJPUR_ENV_SET_EDITOR"
ENV BHOJPUR_ENV_SET_GIT_EDITOR "$BHOJPUR_ENV_SET_EDITOR --wait"
ENV BHOJPUR_ENV_SET_GP_PREVIEW_BROWSER "/ide/bin/bhojpur-code --preview"
ENV BHOJPUR_ENV_SET_GP_EXTERNAL_BROWSER "/ide/bin/bhojpur-code --openExternal"