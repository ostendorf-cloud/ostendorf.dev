<script setup lang="ts">
import { GovueLegalPage, useLanguage } from '@ostendorf-cloud/govue'
import { useRouter } from 'vue-router'

import { operator } from '../legal/operator'

// The site's own privacy text, written from what the code and docs/analytics.md
// state. Anything the repository does not state (hosting provider, who runs
// the Umami instance, retention, legal basis) is deliberately absent, and
// listed in the pull request for the owner. There is no contact form here: the
// site has none.
const { locale } = useLanguage()
const router = useRouter()

function navigate(path: string): void {
  void router.push(path)
}
</script>

<template>
  <GovueLegalPage page="privacy" :skip-link="false" back-href="/" :navigate="navigate">
    <div v-if="locale === 'de'" class="flex flex-col gap-4 text-text">
      <h2 class="text-xl font-semibold">Verantwortlicher</h2>
      <p>
        {{ operator.name }}, {{ operator.street }}, {{ operator.city }}<br />
        E-Mail:
        <a class="text-accent-text hover:underline" :href="`mailto:${operator.email}`">{{
          operator.email
        }}</a>
      </p>

      <h2 class="text-xl font-semibold">Was diese Website auf Ihrem Gerät speichert</h2>
      <p>
        Diese Website setzt keine Cookies und lädt keine Skripte, Schriften oder Bilder von Dritten.
        Wenn Sie über den Schalter in der Kopfzeile die Farbdarstellung (hell, dunkel oder System)
        ändern, speichert Ihr Browser diese Auswahl im lokalen Speicher Ihres Geräts. Sie wird nicht
        an den Server gesendet.
      </p>

      <h2 class="text-xl font-semibold">Zählung der Seitenaufrufe</h2>
      <p>
        Die Website kann Seitenaufrufe auf dem Server mit Umami zählen. Die Zählung ist nur aktiv,
        wenn sie für die Website eingerichtet ist. In Ihrem Browser läuft dafür kein Skript, es wird
        kein Cookie gesetzt und nichts auf Ihrem Gerät gespeichert. Der Aufruf wird gezählt, nachdem
        der Server die Seite ausgeliefert hat.
      </p>
      <p>Bei einem gezählten Aufruf werden an Umami gesendet:</p>
      <ul class="list-disc pl-6">
        <li>
          die Seite als Routenmuster, zum Beispiel <code>/</code>, nicht die aufgerufene Adresse,
        </li>
        <li>die verweisende Website, nur als Ursprung ohne Pfad,</li>
        <li>die erste Sprachangabe Ihres Browsers,</li>
        <li>der User-Agent Ihres Browsers,</li>
        <li>
          Ihre IP-Adresse, vor dem Senden verkürzt: bei IPv4 auf die ersten drei Bytes, bei IPv6 auf
          die ersten sechs Bytes.
        </li>
      </ul>
      <p>
        Aus der verkürzten Adresse leitet Umami das Land und in der Regel die Region ab. Eine
        Abfrage-Zeichenfolge, ein Token oder eine Kennung der aufgerufenen Adresse verlässt den
        Server nicht.
      </p>
      <p>
        Wenn Ihr Browser Do Not Track (<code>DNT: 1</code>) oder Global Privacy Control (<code
          >Sec-GPC: 1</code
        >) sendet, wird Ihr Aufruf nicht gezählt.
      </p>

      <h2 class="text-xl font-semibold">Links zu anderen Websites</h2>
      <p>
        Die Startseite verlinkt auf GitHub, LinkedIn und Stackmeister und nennt eine E-Mail-Adresse.
        Beim Laden der Seite wird keine dieser Seiten kontaktiert. Erst wenn Sie einem Link folgen,
        gelten die Datenschutzhinweise der jeweiligen Website.
      </p>

      <h2 class="text-xl font-semibold">E-Mail</h2>
      <p>
        Wenn Sie mir eine E-Mail schreiben, erhalte ich Ihre E-Mail-Adresse und den Inhalt Ihrer
        Nachricht.
      </p>

      <h2 class="text-xl font-semibold">Ihre Rechte</h2>
      <p>
        Sie haben nach der Datenschutz-Grundverordnung das Recht auf Auskunft, Berichtigung,
        Löschung, Einschränkung der Verarbeitung und Datenübertragbarkeit sowie das Recht, der
        Verarbeitung zu widersprechen. Sie können sich außerdem bei einer
        Datenschutzaufsichtsbehörde beschweren, zum Beispiel in dem Mitgliedstaat Ihres gewöhnlichen
        Aufenthaltsorts, Ihres Arbeitsplatzes oder des Orts des mutmaßlichen Verstoßes. Zur Ausübung
        Ihrer Rechte schreiben Sie an die oben genannte E-Mail-Adresse.
      </p>
    </div>

    <div v-else class="flex flex-col gap-4 text-text">
      <p class="text-text-muted">
        The German version of this page is the legally binding one.
        <a class="text-accent-text hover:underline" href="/de/privacy">View the German version</a>
      </p>

      <h2 class="text-xl font-semibold">Controller</h2>
      <p>
        {{ operator.name }}, {{ operator.street }}, {{ operator.city }}, Germany<br />
        Email:
        <a class="text-accent-text hover:underline" :href="`mailto:${operator.email}`">{{
          operator.email
        }}</a>
      </p>

      <h2 class="text-xl font-semibold">What this site stores on your device</h2>
      <p>
        This site sets no cookies and loads no scripts, fonts or images from third parties. If you
        change the colour mode (light, dark or system) with the switch in the header, your browser
        keeps that choice in the local storage of your device. It is not sent to the server.
      </p>

      <h2 class="text-xl font-semibold">Page view counting</h2>
      <p>
        The site can count page views on the server with Umami. Counting is only active when it is
        set up for the site. No script runs in your browser for it, no cookie is set and nothing is
        stored on your device. A view is counted after the server has delivered the page.
      </p>
      <p>For a counted view, the following is sent to Umami:</p>
      <ul class="list-disc pl-6">
        <li>the page as a route pattern, for example <code>/</code>, not the requested address,</li>
        <li>the referring site, as an origin only, without a path,</li>
        <li>the first language tag of your browser,</li>
        <li>the user agent of your browser,</li>
        <li>
          your IP address, shortened before it is sent: to its first three bytes for IPv4 and its
          first six bytes for IPv6.
        </li>
      </ul>
      <p>
        From the shortened address, Umami derives the country and usually the region. A query
        string, a token or an identifier in the requested address does not leave the server.
      </p>
      <p>
        If your browser sends Do Not Track (<code>DNT: 1</code>) or Global Privacy Control (<code
          >Sec-GPC: 1</code
        >), your view is not counted.
      </p>

      <h2 class="text-xl font-semibold">Links to other websites</h2>
      <p>
        The start page links to GitHub, LinkedIn and Stackmeister and names an email address. None
        of these sites is contacted when the page loads. Only when you follow a link do the privacy
        notices of that website apply.
      </p>

      <h2 class="text-xl font-semibold">Email</h2>
      <p>If you write me an email, I receive your email address and the content of your message.</p>

      <h2 class="text-xl font-semibold">Your rights</h2>
      <p>
        Under the General Data Protection Regulation you have the right of access, rectification,
        erasure, restriction of processing and data portability, and the right to object to
        processing. You can also lodge a complaint with a data protection supervisory authority, for
        example in the member state of your habitual residence, your place of work or the place of
        the alleged infringement. To exercise your rights, write to the email address above.
      </p>
    </div>
  </GovueLegalPage>
</template>
