FROM turbot/steampipe

USER steampipe:0
RUN steampipe plugin install steampipe

# Install the s3 plugin
USER root:0
WORKDIR /home/steampipe
COPY --chown=steampipe:0 build/linux_amd64/steampipe-plugin-s3.plugin .steampipe/plugins/hub.steampipe.io/plugins/Alaffia-Technology-Solutions/s3@latest/steampipe-plugin-s3.plugin
COPY --chown=steampipe:0 config/* .steampipe/config/

# Start steampipe
EXPOSE 9193
USER steampipe:0
ENTRYPOINT ["steampipe"]
CMD ["service", "start", "--foreground", "--database-listen", "network"]