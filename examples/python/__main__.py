import pulumi
import caliban0_pulumi_cherry_servers as cherry

# salt = cherry.RandomSalt("salt", password="test", salted_length=6)
# pulumi.export(
#     "salt",
#     {
#         "salt": salt.salt,
#         "salted_password": salt.salted_password,
#         "password": salt.password,
#         "salt_length": salt.salted_length,
#     },
# )

project = cherry.Project("myProject", team=148226, bgp=True, name="testt")
# pulumi.export(
#     "project_json",
#     pulumi.Output.json_dumps(
#         {
#             "id": project.id,
#             "name": project.name,
#             "team": project.team,
#             "bgp": project.bgp,
#             "local_asn": project.local_asn,
#         }
#     ),
# )
# pulumi.export(
#     "project_output",
#     {
#         "id": project.id,
#         "name": project.name,
#         "team": project.team,
#         "bgp": project.bgp,
#         "local_asn": project.local_asn,
#     },
# )


# ip = cherry.IP(
#     "myIP",
#     region="LT-Siauliai",
#     project=project.id.apply(int),
#     ptr_record="test-a",
#     a_record="test-a",
#     tags={"env": "dev"},
# )
# pulumi.export(
#     "ip_json",
#     pulumi.Output.json_dumps(
#         {
#             "region": ip.region,
#             "project": ip.project,
#             "ptrRecord": ip.ptr_record,
#             "aRecord": ip.a_record,
#             "routedTo": ip.routed_to,
#             "targetedTo": ip.targeted_to,
#             "tags": ip.tags,
#             "address": ip.address,
#             "address_family": ip.address_family,
#             "cidr": ip.cidr,
#             "type": ip.type,
#             "ptrRecordEffective": ip.ptr_record_effective,
#             "aRecordEffective": ip.a_record_effective,
#         }
#     ),
# )

server = cherry.Server(
    "myServer",
    region="LT-Siauliai",
    project=project.id.apply(int),
    plan="B1-1-1gb-20s-shared",
    hostname="hello"
)
pulumi.export(
    "server_output",
    {
        "id": server.id,
        "plan": server.plan,
        "project": server.project,
        "region": server.region,
        "hostname": server.hostname,
        "image": server.image,
        "ssh": server.ssh_keys,
        "extra_ips": server.extra_ips,
        "user_data": server.user_data,
        "tags": server.tags,
        "spot": server.spot,
        "os_partition_size": server.os_partition_size,
        "cycle": server.cycle,
        "discount_code": server.discount_code,
        "block_storage": server.block_storage,
        "bgp": server.bgp,
        "allow_reinstall": server.allow_reinstall,
        "ips": server.ips,
        "pricing": server.pricing,
    },
)

server.id.apply(lambda id: print('Server ID:', id))
server.plan.apply(lambda plan: print('Server plan:', plan))
server.project.apply(lambda project: print('Server project:', project))
server.region.apply(lambda region: print('Server region:', region))
server.hostname.apply(lambda hostname: print('Server hostname:', hostname))
server.image.apply(lambda image: print('Server image:', image))
server.ssh_keys.apply(lambda ssh: print('Server ssh:', ssh))
server.extra_ips.apply(lambda extra_ips: print('Server extra IPs:', extra_ips))
server.user_data.apply(lambda user_data: print('Server user data:', user_data))
server.tags.apply(lambda tags: print('Server tags:', tags))
server.spot.apply(lambda spot: print('Server spot:', spot))
server.os_partition_size.apply(lambda os: print('Server OS partition size:', os))
server.cycle.apply(lambda cycle: print('Server cycle:', cycle))
server.discount_code.apply(lambda discount: print('Server discount code:', discount))
server.block_storage.apply(lambda block: print('Server block storage:', block))
server.bgp.apply(lambda bgp: print('Server BGP:', bgp))
server.allow_reinstall.apply(lambda allow_reinstall: print('Server allow reinstall:', allow_reinstall))
server.ips.apply(lambda ips: print('Server IPs:', ips))
server.pricing.apply(lambda pricing: print('Server pricing:', pricing))
