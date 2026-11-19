<script setup lang="ts">
import { GovueLegalPage, useLanguage } from '@ostendorf-cloud/govue'

import { operator } from '../legal/operator'

// The site's own privacy text, written from what the code and docs/analytics.md
// state. Hosting, retention and legal basis are the operator's own facts,
// stated by the owner, not properties of the code. There is no contact form
// here: the site has none.
// No navigate prop: the language switch is a full page load, every language
// is its own prerendered page with its own head.
const { locale } = useLanguage()
</script>

<template>
  <GovueLegalPage page="privacy" :skip-link="false" back-href="/">
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

      <h2 class="text-xl font-semibold">Betrieb der Website</h2>
      <p>
        Die Website wird auf eigener Hardware in Deutschland betrieben. Einen Hosting-Anbieter
        setzen wir dafür nicht ein.
      </p>

      <h2 class="text-xl font-semibold">Zugriffsprotokolle</h2>
      <p>
        Beim Aufruf der Website schreiben die Anwendung und der vorgeschaltete Ingress
        Zugriffsprotokolle, die auch Ihre IP-Adresse enthalten. Sie dienen dem sicheren und stabilen
        Betrieb. Die Protokolle werden nach 30 Tagen gelöscht. Rechtsgrundlage ist Art. 6 Abs. 1
        lit. f DSGVO, unser berechtigtes Interesse am sicheren und stabilen Betrieb der Website.
      </p>

      <h2 class="text-xl font-semibold">Zählung der Seitenaufrufe</h2>
      <p>
        Die Website zählt Seitenaufrufe auf dem Server mit Umami. Wir betreiben Umami selbst, auf
        eigener Hardware in Deutschland. In Ihrem Browser läuft dafür kein Skript, es wird kein
        Cookie gesetzt und nichts auf Ihrem Gerät gespeichert. Der Aufruf wird gezählt, nachdem der
        Server die Seite ausgeliefert hat.
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
        Aus der verkürzten Adresse leitet Umami Land, Region und Stadt ab, aus dem User-Agent
        Browser und Betriebssystem. Die vollständige IP-Adresse und der vollständige User-Agent
        werden nicht gespeichert. Durch die Kürzung ist die Stadt oft ungenau. Eine
        Abfrage-Zeichenfolge, ein Token oder eine Kennung der aufgerufenen Adresse verlässt den
        Server nicht.
      </p>
      <p>
        Die Statistikdaten werden nach 14 Monaten gelöscht. Rechtsgrundlage ist Art. 6 Abs. 1 lit. f
        DSGVO, unser berechtigtes Interesse an der sicheren und stabilen Bereitstellung der Website
        und an der anonymen Messung der Reichweite.
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

      <h2 class="text-xl font-semibold">Operation of the site</h2>
      <p>The site is operated on our own hardware in Germany. We use no hosting provider for it.</p>

      <h2 class="text-xl font-semibold">Access logs</h2>
      <p>
        When you visit the site, the application and the ingress in front of it write access logs
        that also contain your IP address. They serve secure and stable operation. The logs are
        deleted after 30 days. The legal basis is Art. 6(1)(f) GDPR, our legitimate interest in the
        secure and stable operation of the site.
      </p>

      <h2 class="text-xl font-semibold">Page view counting</h2>
      <p>
        The site counts page views on the server with Umami. We operate Umami ourselves, on our own
        hardware in Germany. No script runs in your browser for it, no cookie is set and nothing is
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
        From the shortened address, Umami derives the country, region and city, and from the user
        agent the browser and the operating system. The full IP address and the full user agent are
        not stored. The shortening makes the city less accurate. A query string, a token or an
        identifier in the requested address does not leave the server.
      </p>
      <p>
        The statistics data is deleted after 14 months. The legal basis is Art. 6(1)(f) GDPR, our
        legitimate interest in the secure and stable provision of the site and in anonymous
        measurement of reach.
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
