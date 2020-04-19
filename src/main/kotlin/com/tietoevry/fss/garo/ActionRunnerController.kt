package com.tietoevry.fss.garo

import com.fasterxml.jackson.module.kotlin.registerKotlinModule
import com.tietoevry.fss.garo.crd.ActionRunner
import com.tietoevry.fss.garo.crd.ActionRunnerList
import com.tietoevry.fss.garo.github.Runners
import io.fabric8.kubernetes.api.model.Pod
import io.fabric8.kubernetes.api.model.PodBuilder
import io.fabric8.kubernetes.api.model.PodList
import io.fabric8.kubernetes.client.KubernetesClient
import io.fabric8.kubernetes.client.dsl.base.CustomResourceDefinitionContext
import io.fabric8.kubernetes.client.informers.ResourceEventHandler
import io.fabric8.kubernetes.client.informers.SharedIndexInformer
import io.fabric8.kubernetes.client.informers.cache.Lister
import io.fabric8.kubernetes.client.utils.Serialization
import mu.KotlinLogging
import java.nio.charset.StandardCharsets
import java.time.Duration
import java.util.*
import java.util.concurrent.ArrayBlockingQueue
import java.util.concurrent.BlockingQueue
import javax.ws.rs.client.Client
import javax.ws.rs.client.WebTarget
import javax.ws.rs.core.MediaType

class ActionRunnerController(webClient: Client,
                             private val kubernetesClient: KubernetesClient ) {

    private val logger = KotlinLogging.logger {}
    private val blockingQueue: BlockingQueue<ActionRunner> = ArrayBlockingQueue(1024)

    private val podSharedInformer: SharedIndexInformer<Pod>
    private val podLister: Lister<Pod>
    private val actionRunnerSharedIndexInformer: SharedIndexInformer<ActionRunner>
    private val customResourceDefinitionContext: CustomResourceDefinitionContext
    private val githubApi: WebTarget

    init {
        Serialization.jsonMapper().registerKotlinModule()

        this.customResourceDefinitionContext = CustomResourceDefinitionContext.Builder()
                .withScope("Namespaced")
                .withGroup("garo.tietoevry.com")
                .withName("ActionRunner")
                .withPlural("actionrunners")
                .withVersion("v1alpha1")
                .build()

        val sharedInformerFactory = kubernetesClient.informers()
        val resyncPeriod = Duration.ofMinutes(1)
        this.actionRunnerSharedIndexInformer = sharedInformerFactory
                .sharedIndexInformerForCustomResource(customResourceDefinitionContext,
                        ActionRunner::class.java, ActionRunnerList::class.java, resyncPeriod.toMillis() )
        this.podSharedInformer = sharedInformerFactory.sharedIndexInformerFor(Pod::class.java, PodList::class.java, resyncPeriod.toMillis())
        this.podLister = Lister(podSharedInformer.indexer, kubernetesClient.namespace)

        this.actionRunnerSharedIndexInformer.addEventHandler( object : ResourceEventHandler<ActionRunner> {
            override fun onAdd(obj: ActionRunner) {
                logger.info { "add $obj" }
                blockingQueue.add(obj)
            }

            override fun onDelete(obj: ActionRunner, deletedFinalStateUnknown: Boolean) {
                // NOOP
            }

            override fun onUpdate(oldObj: ActionRunner, newObj: ActionRunner) {
                blockingQueue.add(newObj)
            }
        })

        this.githubApi = webClient.target("https://api.github.com")

        sharedInformerFactory.startAllRegisteredInformers()
    }

    fun controlLoop() {
        while (!podSharedInformer.hasSynced() || !actionRunnerSharedIndexInformer.hasSynced() ) {
            logger.info { "Waiting for informer sync..." }
            Thread.sleep(1000)
        }

        logger.info { "Informers synced" }

        while (true) {
            try {
                blockingQueue.take().also { reconcile(it) }
            }
            catch ( e: Exception ) {
                logger.error(e.message, e)
            }
        }
    }

    private fun reconcile(actionRunner: ActionRunner) {
        logger.debug { "Reconciling $actionRunner" }

        // TODO: Should be cached
        val token = kubernetesClient.secrets()
                .inNamespace(actionRunner.metadata.namespace)
                .withName(actionRunner.spec.tokenRef.name)
                .get()
                .data[actionRunner.spec.tokenRef.key]
                .let { String(Base64.getDecoder().decode(it), StandardCharsets.UTF_8) }

        // TODO: listing needs filtering to tie them to this specific runner spec
        val runners = this.githubApi.path("/orgs/${actionRunner.spec.organization}/actions/runners")
            .request()
            .header("Authorization", "token $token")
            .accept(MediaType.APPLICATION_JSON)
            .get(Runners::class.java)
        if ( runners.totalCount < actionRunner.spec.minRunners &&
            listRelatedPods(actionRunner).size == runners.totalCount /* all have registered */) {
            createBuildPod(actionRunner)
        }
        else if ( runners.totalCount > actionRunner.spec.maxRunners ) {
            listRelatedPods(actionRunner)
                .subList(0, runners.totalCount-actionRunner.spec.maxRunners)
                .let { kubernetesClient.pods().delete(it) }
        }
    }

    private fun listRelatedPods(actionRunner: ActionRunner): List<Pod> =
        podLister.list().filter { pod -> pod.metadata.ownerReferences.contains(actionRunner.asOwnerRef()) }

    private fun createBuildPod(actionRunner: ActionRunner) {
        val pod = PodBuilder()
            .withNewMetadata()
                .withNamespace(actionRunner.metadata.namespace)
                .withGenerateName(actionRunner.metadata.name + "-")
                .withLabels(mapOf( "app" to actionRunner.metadata.name ))
                .addToOwnerReferences(actionRunner.asOwnerRef())
            .endMetadata()
                .withNewSpecLike(actionRunner.spec.podSpec)
                .endSpec()
            .build()
        this.kubernetesClient.pods().inNamespace(actionRunner.metadata.namespace).create(pod)
    }

}