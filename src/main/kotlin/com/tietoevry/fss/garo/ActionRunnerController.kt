package com.tietoevry.fss.garo

import com.fasterxml.jackson.module.kotlin.registerKotlinModule
import com.tietoevry.fss.garo.crd.ActionRunner
import com.tietoevry.fss.garo.crd.ActionRunnerList
import com.tietoevry.fss.garo.github.RunnerApi
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
import java.time.Duration
import java.util.concurrent.ArrayBlockingQueue
import java.util.concurrent.BlockingQueue

class ActionRunnerController(private val runnerApi: RunnerApi,
                             private val kubernetesClient: KubernetesClient ) {

    private val logger = KotlinLogging.logger {}
    private val blockingQueue: BlockingQueue<ActionRunner> = ArrayBlockingQueue(1024)

    private val podSharedInformer: SharedIndexInformer<Pod>
    private val podLister: Lister<Pod>
    private val actionRunnerSharedIndexInformer: SharedIndexInformer<ActionRunner>
    private val customResourceDefinitionContext: CustomResourceDefinitionContext

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
                logger.info { "Delete $obj" }
            }

            override fun onUpdate(oldObj: ActionRunner, newObj: ActionRunner) {
                blockingQueue.add(newObj)
            }
        })

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
                val githubRunner = blockingQueue.take()
                reconcile(githubRunner)
            }
            catch ( e: Exception ) {
                logger.error(e.message, e)
            }
        }
    }

    private fun reconcile(actionRunner: ActionRunner) {
        logger.debug { "Reconciling $actionRunner" }
        val token = System.getenv("GH_TOKEN")
        // TODO: listing needs filtering to tie them to this specific runner spec
        val runners = runnerApi.listOrgRunners("token $token", actionRunner.spec.organization)
        if ( runners.totalCount < actionRunner.spec.minRunners &&
                listRelatedPods(actionRunner).size == runners.totalCount /* all have registered */) {
            createBuildPod(actionRunner)
        }
        val busyRunners = runners.runners.filter { r -> r.status == "busy" }.size
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