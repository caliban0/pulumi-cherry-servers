import pulumi
import caliban0_pulumi_cherry_servers as cherry

project = cherry.Project("myProject", name="myProject", team=148226)
pulumi.export("output", {
    "value": project.bgp,
})