from jinja2 import Template
import sys,os

templateText=r"""
node:
  cluster: test-cluster
  id: test-id

admin:
  access_log_path: /dev/null
  address:
    socket_address:
      address: 0.0.0.0
      port_value: 9901

dynamic_resources:
  cds_config:
    resource_api_version: V3
    api_config_source:
      api_type: GRPC
      transport_api_version: V3
      grpc_services:
        - envoy_grpc:
            cluster_name: xds_cluster
      set_node_on_first_message_only: true
static_resources:
  listeners:
  - name: listener_http
    address:
      socket_address: { address: 0.0.0.0, port_value: 80 }
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
          codec_type: AUTO
          stat_prefix: ingress_http
          route_config:
            name: local_route
            virtual_hosts:
            - name: backend
              domains:
              - "{{DOMAIN_NAME}}"
              routes:
              - match:
                  prefix: "/"
                redirect:
                  https_redirect: true
          http_filters:
          - name: envoy.filters.http.router

  - name: listener_https
    address:
      socket_address: { address: 0.0.0.0, port_value: 443 }
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
          codec_type: AUTO
          stat_prefix: ingress_http
          rds:
            config_source:
              resource_api_version: V3
              api_config_source:
                api_type: gRPC
                transport_api_version: V3
                grpc_services:
                  - envoy_grpc:
                      cluster_name: xds_cluster
                set_node_on_first_message_only: true
            route_config_name: discovered_container_services
          http_filters:
          - name: envoy.filters.http.router

      transport_socket:
        name: envoy.transport_sockets.tls
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.DownstreamTlsContext
          common_tls_context:
            tls_certificates:
            - certificate_chain:
                filename: "/etc/letsencrypt/live/{{DOMAIN_NAME}}/fullchain.pem"
              private_key:
                filename: "/etc/letsencrypt/live/{{DOMAIN_NAME}}/privkey.pem"

  clusters:
    - connect_timeout: 1s
      type: STATIC
      load_assignment:
        cluster_name: xds_cluster
        endpoints:
          - lb_endpoints:
              - endpoint:
                  address:
                    socket_address:
                      address: REPlACE_HOSTADDRESS
                      port_value: 18000
      http2_protocol_options: {}
      name: xds_cluster
layered_runtime:
  layers:
    - name: runtime-0
      rtds_layer:
        rtds_config:
          resource_api_version: V3
          api_config_source:
            transport_api_version: V3
            api_type: GRPC
            grpc_services:
              envoy_grpc:
                cluster_name: xds_cluster
        name: runtime-0
"""

def render_template(templateText, templateTextFilePath):
    template = Template(templateText)
    for k, v in os.environ.items():
        template.globals[k]=v
    currentAbsPath=os.path.abspath(templateTextFilePath)
    path,filename=os.path.split(currentAbsPath)
    renderedAbsPath = os.path.join(path, filename.replace("-template.py",".yaml"))
    file =open(renderedAbsPath,"w")
    print('content of {} : '.format(renderedAbsPath))
    content=template.render()
    print(content)
    file.write(content)

render_template(templateText,__file__)