import pulumi
import caliban0_pulumi_cherry_servers as cherry

project = cherry.Project("myProject", team=148226, name="myProject", bgp=False)
pulumi.export("output", {
    "bgp": project.bgp, "name": project.name, "team": project.team
})